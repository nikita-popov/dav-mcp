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

func RegisterJournal(s *mcp.Server, cfg config.Config) {

	// calendar_journal_list
	s.AddTool(
		"calendar_journal_list",
		"List VJOURNAL entries from a calendar. Optionally filter by status: DRAFT, FINAL, CANCELLED.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"calendar": {Type: "string", Description: "Calendar path from calendar_calendar_list (optional, uses primary calendar if omitted)"},
				"status":   {Type: "string", Description: "Filter by STATUS: DRAFT, FINAL, CANCELLED (optional, returns all if omitted)"},
				"account":  {Type: "string", Description: "Account name (optional)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if !sess.Caps.Supports("VJOURNAL") {
				return nil, fmt.Errorf("server does not support VJOURNAL")
			}

			calPath := journalCalPath(args, sess)
			if calPath == "" {
				return nil, fmt.Errorf("no calendars found in session")
			}

			blobs, err := dav.QueryJournals(ctx, sess.Client, calPath)
			if err != nil {
				return nil, err
			}

			var journals []ical.ParsedJournal
			for _, blob := range blobs {
				journals = append(journals, ical.ParseJournals(blob)...)
			}

			statusFilter := strings.ToUpper(strArg(args, "status"))
			if statusFilter != "" {
				var filtered []ical.ParsedJournal
				for _, j := range journals {
					if j.Status == statusFilter {
						filtered = append(filtered, j)
					}
				}
				journals = filtered
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatJournals(journals),
				}},
			}, nil
		},
	)

	// calendar_journal_get
	s.AddTool(
		"calendar_journal_get",
		"Get a single VJOURNAL entry by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Journal UID"},
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

			ref, err := findJournalByUID(ctx, sess, strArg(args, "uid"), strArg(args, "calendar"))
			if err != nil {
				return nil, err
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatJournal(ref.rec.Journal),
				}},
			}, nil
		},
	)

	// calendar_journal_create
	s.AddTool(
		"calendar_journal_create",
		"Create a new VJOURNAL entry. Returns the UID of the created journal.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Journal title"},
				"description": {Type: "string", Description: "Journal body text (optional)"},
				"date":        {Type: "string", Description: "Journal date, ISO 8601 date or datetime (optional, defaults to today)"},
				"status":      {Type: "string", Description: "Initial status: DRAFT, FINAL (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path from calendar_calendar_list (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"summary"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary"},
				Optional: []string{"description", "date", "status", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}
			if !sess.Caps.Supports("VJOURNAL") {
				return nil, fmt.Errorf("server does not support VJOURNAL")
			}

			calPath := journalCalPath(args, sess)
			if calPath == "" {
				return nil, fmt.Errorf("no calendars found in session")
			}

			date := time.Now().UTC()
			if v := strArg(args, "date"); v != "" {
				if t, err2 := time.Parse(time.RFC3339, v); err2 == nil {
					date = t.UTC()
				} else if t, err2 := time.Parse("2006-01-02", v); err2 == nil {
					date = t.UTC()
				} else {
					return nil, fmt.Errorf("invalid date %q: use ISO 8601", v)
				}
			}

			j := ical.Journal{
				Summary:     strArg(args, "summary"),
				Description: strArg(args, "description"),
				Date:        date,
				Status:      strings.ToUpper(strArg(args, "status")),
			}
			icsData := ical.BuildJournal(j)
			parsed := ical.ParseJournals(icsData)
			uid := ""
			if len(parsed) > 0 {
				uid = parsed[0].UID
			}

			if err := dav.PutJournal(ctx, sess.Client, calPath, uid, icsData, ""); err != nil {
				return nil, fmt.Errorf("calendar_journal_create: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Journal created.\nUID: %s\nCalendar: %s", uid, calPath),
				}},
			}, nil
		},
	)

	// calendar_journal_update
	s.AddTool(
		"calendar_journal_update",
		"Update an existing VJOURNAL entry by UID. Only supplied fields are changed.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Journal UID (required)"},
				"summary":     {Type: "string", Description: "New title (optional)"},
				"description": {Type: "string", Description: "New body text (optional)"},
				"date":        {Type: "string", Description: "New date, ISO 8601 (optional)"},
				"status":      {Type: "string", Description: "New status: DRAFT, FINAL, CANCELLED (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional, searches all if omitted)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"summary", "description", "date", "status", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			uid := strArg(args, "uid")
			ref, err := findJournalByUID(ctx, sess, uid, strArg(args, "calendar"))
			if err != nil {
				return nil, err
			}

			j := ical.Journal{
				UID:         ref.rec.Journal.UID,
				Summary:     ref.rec.Journal.Summary,
				Description: ref.rec.Journal.Description,
				Date:        ref.rec.Journal.Date,
				Status:      ref.rec.Journal.Status,
			}
			if v := strArg(args, "summary"); v != "" {
				j.Summary = v
			}
			if _, ok := args["description"]; ok {
				j.Description = strArg(args, "description")
			}
			if v := strArg(args, "date"); v != "" {
				if t, err2 := time.Parse(time.RFC3339, v); err2 == nil {
					j.Date = t.UTC()
				} else if t, err2 := time.Parse("2006-01-02", v); err2 == nil {
					j.Date = t.UTC()
				} else {
					return nil, fmt.Errorf("invalid date %q: use ISO 8601", v)
				}
			}
			if v := strArg(args, "status"); v != "" {
				j.Status = strings.ToUpper(v)
			}

			icsData := ical.BuildJournal(j)
			if err := dav.PutJournalHref(ctx, sess.Client, ref.rec.Href, icsData, ref.rec.ETag); err != nil {
				return nil, fmt.Errorf("calendar_journal_update: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Journal updated.\nUID: %s", uid),
				}},
			}, nil
		},
	)

	// calendar_journal_delete
	s.AddTool(
		"calendar_journal_delete",
		"Delete a VJOURNAL entry by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Journal UID"},
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
			ref, err := findJournalByUID(ctx, sess, uid, strArg(args, "calendar"))
			if err != nil {
				return nil, fmt.Errorf("calendar_journal_delete: %w", err)
			}

			if err := dav.DeleteJournal(ctx, sess.Client, ref.rec.Href, ref.rec.ETag); err != nil {
				return nil, fmt.Errorf("calendar_journal_delete: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Deleted journal UID=%s", uid),
				}},
			}, nil
		},
	)
}

// ---- helpers ----------------------------------------------------------------

type journalRef struct {
	rec          *dav.JournalRecord
	calendarHref string
}

func findJournalByUID(ctx context.Context, sess *dav.Session, uid, calendarHref string) (*journalRef, error) {
	calendars := sess.Calendars
	if calendarHref != "" {
		calendars = []dav.Collection{{Href: calendarHref}}
	}
	for _, cal := range calendars {
		rec, err := dav.QueryJournalByUID(ctx, sess.Client, cal.Href, uid)
		if err != nil {
			continue
		}
		return &journalRef{rec: rec, calendarHref: cal.Href}, nil
	}
	return nil, fmt.Errorf("journal UID=%q not found", uid)
}

// journalCalPath returns calendar path from args or the first VJOURNAL-capable calendar.
func journalCalPath(args map[string]any, sess *dav.Session) string {
	if v := strArg(args, "calendar"); v != "" {
		return v
	}
	for _, cal := range sess.Calendars {
		if cal.Supports("VJOURNAL") {
			return cal.Href
		}
	}
	if len(sess.Calendars) > 0 {
		return sess.Calendars[0].Href
	}
	return ""
}

func formatJournals(journals []ical.ParsedJournal) string {
	if len(journals) == 0 {
		return "No journal entries found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d journal entry(s):\n\n", len(journals))
	for _, j := range journals {
		b.WriteString(formatJournal(j))
		b.WriteString("\n")
	}
	return b.String()
}

func formatJournal(j ical.ParsedJournal) string {
	var b strings.Builder
	fmt.Fprintf(&b, "UID: %s\n", j.UID)
	fmt.Fprintf(&b, "Summary: %s\n", j.Summary)
	if !j.Date.IsZero() {
		fmt.Fprintf(&b, "Date: %s\n", j.Date.Format(time.DateOnly))
	}
	if j.Status != "" {
		fmt.Fprintf(&b, "Status: %s\n", j.Status)
	}
	if j.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", j.Description)
	}
	return b.String()
}
