package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/ical"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

func RegisterTodo(s *mcp.Server, cfg config.Config) {

	// calendar_todo_list
	s.AddTool(
		"calendar_todo_list",
		"List VTODO items from a calendar. Optionally filter by status: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"calendar": {Type: "string", Description: "Calendar path from calendar_calendar_list (optional, uses primary calendar if omitted)"},
				"status":   {Type: "string", Description: "Filter by STATUS: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED (optional, returns all if omitted)"},
				"account":  {Type: "string", Description: "Account name (optional)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if !sess.Caps.Supports("VTODO") {
				return nil, fmt.Errorf("server does not support VTODO")
			}

			calPath := todoCalPath(args, sess)
			if calPath == "" {
				return nil, fmt.Errorf("no calendars found in session")
			}

			blobs, err := dav.QueryTodos(ctx, sess.Client, calPath)
			if err != nil {
				return nil, err
			}

			var todos []ical.ParsedTodo
			for _, blob := range blobs {
				todos = append(todos, ical.ParseTodos(blob)...)
			}

			statusFilter := strings.ToUpper(strArg(args, "status"))
			if statusFilter != "" {
				var filtered []ical.ParsedTodo
				for _, t := range todos {
					if t.Status == statusFilter {
						filtered = append(filtered, t)
					}
				}
				todos = filtered
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatTodos(todos),
				}},
			}, nil
		},
	)

	// calendar_todo_get
	s.AddTool(
		"calendar_todo_get",
		"Get a single VTODO item by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Todo UID"},
				"calendar": {Type: "string", Description: "Calendar path (optional, searches all calendars if omitted)"},
				"account":  {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			ref, err := findTodoByUID(ctx, sess, strArg(args, "uid"), strArg(args, "calendar"))
			if err != nil {
				return nil, err
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatTodo(ref.rec.Todo),
				}},
			}, nil
		},
	)

	// calendar_todo_create
	s.AddTool(
		"calendar_todo_create",
		"Create a new VTODO item. Returns the UID of the created todo.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Todo title"},
				"description": {Type: "string", Description: "Details (optional)"},
				"due":         {Type: "string", Description: "Due datetime, ISO 8601 (optional)"},
				"priority":    {Type: "number", Description: "Priority 1 (highest) – 9 (lowest), 0 = undefined (optional)"},
				"status":      {Type: "string", Description: "Initial status: NEEDS-ACTION, IN-PROCESS (optional, defaults to NEEDS-ACTION)"},
				"calendar":    {Type: "string", Description: "Calendar path from calendar_calendar_list (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"summary"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary"},
				Optional: []string{"description", "due", "priority", "status", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if !sess.Caps.Supports("VTODO") {
				return nil, fmt.Errorf("server does not support VTODO")
			}

			calPath := todoCalPath(args, sess)
			if calPath == "" {
				return nil, fmt.Errorf("no calendars found in session")
			}

			todo := ical.Todo{
				Summary:     strArg(args, "summary"),
				Description: strArg(args, "description"),
				Status:      strings.ToUpper(strArg(args, "status")),
			}
			if v := strArg(args, "due"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return nil, fmt.Errorf("invalid due: %w", err)
				}
				todo.Due = t.UTC()
			}
			if p, ok := args["priority"].(float64); ok {
				todo.Priority = int(p)
			}

			icsData := ical.BuildTodo(todo)
			parsed := ical.ParseTodos(icsData)
			uid := ""
			if len(parsed) > 0 {
				uid = parsed[0].UID
			}

			if err := dav.PutTodo(ctx, sess.Client, calPath, uid, icsData, ""); err != nil {
				return nil, fmt.Errorf("calendar_todo_create: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Todo created.\nUID: %s\nCalendar: %s", uid, calPath),
				}},
			}, nil
		},
	)

	// calendar_todo_update
	s.AddTool(
		"calendar_todo_update",
		"Update an existing VTODO by UID. Only supplied fields are changed.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Todo UID (required)"},
				"summary":     {Type: "string", Description: "New title (optional)"},
				"description": {Type: "string", Description: "New description (optional)"},
				"due":         {Type: "string", Description: "New due datetime, ISO 8601 (optional)"},
				"priority":    {Type: "number", Description: "New priority 1–9, 0 = clear (optional)"},
				"status":      {Type: "string", Description: "New status: NEEDS-ACTION, IN-PROCESS, COMPLETED, CANCELLED (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional, searches all if omitted)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"summary", "description", "due", "priority", "status", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			uid := strArg(args, "uid")
			ref, err := findTodoByUID(ctx, sess, uid, strArg(args, "calendar"))
			if err != nil {
				return nil, err
			}

			todo := parsedToTodo(ref.rec.Todo)
			if v := strArg(args, "summary"); v != "" {
				todo.Summary = v
			}
			if _, ok := args["description"]; ok {
				todo.Description = strArg(args, "description")
			}
			if v := strArg(args, "due"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return nil, fmt.Errorf("invalid due: %w", err)
				}
				todo.Due = t.UTC()
			}
			if p, ok := args["priority"].(float64); ok {
				todo.Priority = int(p)
			}
			if v := strArg(args, "status"); v != "" {
				todo.Status = strings.ToUpper(v)
			}

			icsData := ical.BuildTodo(todo)
			if err := dav.PutTodoHref(ctx, sess.Client, ref.rec.Href, icsData, ref.rec.ETag); err != nil {
				return nil, fmt.Errorf("calendar_todo_update: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Todo updated.\nUID: %s", uid),
				}},
			}, nil
		},
	)

	// calendar_todo_delete
	s.AddTool(
		"calendar_todo_delete",
		"Delete a VTODO item by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Todo UID"},
				"calendar": {Type: "string", Description: "Calendar path (optional, searches all if omitted)"},
				"account":  {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			uid := strArg(args, "uid")
			ref, err := findTodoByUID(ctx, sess, uid, strArg(args, "calendar"))
			if err != nil {
				return nil, fmt.Errorf("calendar_todo_delete: %w", err)
			}

			if err := dav.DeleteTodo(ctx, sess.Client, ref.rec.Href, ref.rec.ETag); err != nil {
				return nil, fmt.Errorf("calendar_todo_delete: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Deleted todo UID=%s", uid),
				}},
			}, nil
		},
	)
}

// ---- helpers ----------------------------------------------------------------

type todoRef struct {
	rec          *dav.TodoRecord
	calendarHref string
}

func findTodoByUID(ctx context.Context, sess *dav.Session, uid, calendarHref string) (*todoRef, error) {
	calendars := sess.Calendars
	if calendarHref != "" {
		calendars = []dav.Collection{{Href: calendarHref}}
	}
	for _, cal := range calendars {
		rec, err := dav.QueryTodoByUID(ctx, sess.Client, cal.Href, uid)
		if err != nil {
			continue
		}
		return &todoRef{rec: rec, calendarHref: cal.Href}, nil
	}
	return nil, fmt.Errorf("todo UID=%q not found", uid)
}

// parsedToTodo converts ParsedTodo → ical.Todo for BuildTodo.
func parsedToTodo(p ical.ParsedTodo) ical.Todo {
	return ical.Todo{
		UID:         p.UID,
		Summary:     p.Summary,
		Description: p.Description,
		Due:         p.Due,
		Priority:    p.Priority,
		Status:      p.Status,
	}
}

// todoCalPath returns calendar path from args or the first VTODO-capable calendar.
func todoCalPath(args map[string]any, sess *dav.Session) string {
	if v := strArg(args, "calendar"); v != "" {
		return v
	}
	for _, cal := range sess.Calendars {
		if cal.Supports("VTODO") {
			return cal.Href
		}
	}
	if len(sess.Calendars) > 0 {
		return sess.Calendars[0].Href
	}
	return ""
}

func formatTodos(todos []ical.ParsedTodo) string {
	if len(todos) == 0 {
		return "No todos found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d todo(s):\n\n", len(todos))
	for _, t := range todos {
		b.WriteString(formatTodo(t))
		b.WriteString("\n")
	}
	return b.String()
}

func formatTodo(t ical.ParsedTodo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "UID: %s\n", t.UID)
	fmt.Fprintf(&b, "Summary: %s\n", t.Summary)
	if t.Status != "" {
		fmt.Fprintf(&b, "Status: %s\n", t.Status)
	}
	if !t.Due.IsZero() {
		fmt.Fprintf(&b, "Due: %s\n", t.Due.Format(time.RFC3339))
	}
	if t.Priority > 0 {
		fmt.Fprintf(&b, "Priority: %d\n", t.Priority)
	}
	if t.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", t.Description)
	}
	return b.String()
}
