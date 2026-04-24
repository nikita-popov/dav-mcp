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

func RegisterCalendar(s *mcp.Server, cfg config.Config) {

	// calendar_connect
	s.AddTool(
		"calendar_connect",
		"Connect to a CalDAV server and discover calendars. Returns a list of available calendars.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"url":      {Type: "string", Description: "CalDAV server URL"},
				"username": {Type: "string", Description: "Username"},
				"password": {Type: "string", Description: "Password"},
				"account":  {Type: "string", Description: "Account name to store this connection under (optional, defaults to \"default\")"},
			},
			Required: []string{"url", "username", "password"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"url", "username", "password"},
				Optional: []string{"account"},
			}, args); err != nil {
				return nil, err
			}
			rawURL, _ := args["url"].(string)
			username, _ := args["username"].(string)
			password, _ := args["password"].(string)
			accName := strArg(args, "account")
			if accName == "" {
				accName = "default"
			}

			sess, err := dav.Connect(ctx, accName, rawURL, username, password)
			if err != nil {
				return nil, err
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatCalendars(accName, sess),
				}},
			}, nil
		},
	)

	// calendar_reconnect
	s.AddTool(
		"calendar_reconnect",
		"Reconnect one or all accounts using credentials from environment variables (DAV_URL / DAV_ACCOUNTS).",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"account": {Type: "string", Description: "Account name to reconnect (optional, reconnects all if omitted)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			accName := strArg(args, "account")

			accounts := cfg.Accounts
			if accName != "" {
				acc, err := cfg.Account(accName)
				if err != nil {
					return nil, err
				}
				accounts = []config.Account{acc}
			}
			if len(accounts) == 0 {
				return nil, fmt.Errorf("no accounts configured")
			}

			var b strings.Builder
			for _, acc := range accounts {
				sess, err := dav.Connect(ctx, acc.Name, acc.URL, acc.Username, acc.Password)
				if err != nil {
					fmt.Fprintf(&b, "account %q: connect error: %v\n", acc.Name, err)
					continue
				}
				b.WriteString(formatCalendars(acc.Name, sess))
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{Type: "text", Text: b.String()}},
			}, nil
		},
	)

	// calendar_get_events
	s.AddTool(
		"calendar_get_events",
		"List calendar events in a time range.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"start":    {Type: "string", Description: "Range start, ISO 8601, e.g. 2026-04-01T00:00:00Z"},
				"end":      {Type: "string", Description: "Range end, ISO 8601, e.g. 2026-04-30T23:59:59Z"},
				"calendar": {Type: "string", Description: "Calendar path (optional, defaults to primary discovered calendar)"},
				"account":  {Type: "string", Description: "Account name (optional, defaults to primary account)"},
			},
			Required: []string{"start", "end"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"start", "end"},
				Optional: []string{"calendar", "account"},
			}, args); err != nil {
				return nil, err
			}

			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			startStr, _ := args["start"].(string)
			endStr, _ := args["end"].(string)

			startT, err := time.Parse(time.RFC3339, startStr)
			if err != nil {
				return nil, fmt.Errorf("invalid start: %w", err)
			}
			endT, err := time.Parse(time.RFC3339, endStr)
			if err != nil {
				return nil, fmt.Errorf("invalid end: %w", err)
			}

			calPath, _ := args["calendar"].(string)
			if calPath == "" {
				if len(sess.Calendars) == 0 {
					return nil, fmt.Errorf("no calendars found in session")
				}
				calPath = sess.Calendars[0].Href
			}

			const icalFmt = "20060102T150405Z"
			blobs, err := dav.QueryEvents(ctx, sess.Client, calPath,
				startT.UTC().Format(icalFmt),
				endT.UTC().Format(icalFmt),
			)
			if err != nil {
				return nil, err
			}

			var allEvents []ical.ParsedEvent
			for _, blob := range blobs {
				allEvents = append(allEvents, ical.ParseEvents(blob)...)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatEvents(allEvents, startT, endT),
				}},
			}, nil
		},
	)

	// calendar_create_event
	s.AddTool(
		"calendar_create_event",
		"Create a new calendar event. Returns the UID of the created event.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Event title"},
				"start":       {Type: "string", Description: "Start datetime, ISO 8601, e.g. 2026-05-01T10:00:00Z"},
				"end":         {Type: "string", Description: "End datetime, ISO 8601"},
				"description": {Type: "string", Description: "Event description (optional)"},
				"location":    {Type: "string", Description: "Location (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional, defaults to primary)"},
				"account":     {Type: "string", Description: "Account name (optional, defaults to primary account)"},
			},
			Required: []string{"summary", "start", "end"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary", "start", "end"},
				Optional: []string{"description", "location", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}

			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			summary, _ := args["summary"].(string)
			startStr, _ := args["start"].(string)
			endStr, _ := args["end"].(string)
			desc, _ := args["description"].(string)
			loc, _ := args["location"].(string)

			startT, err := time.Parse(time.RFC3339, startStr)
			if err != nil {
				return nil, fmt.Errorf("invalid start: %w", err)
			}
			endT, err := time.Parse(time.RFC3339, endStr)
			if err != nil {
				return nil, fmt.Errorf("invalid end: %w", err)
			}

			calPath, _ := args["calendar"].(string)
			if calPath == "" {
				if len(sess.Calendars) == 0 {
					return nil, fmt.Errorf("no calendars found in session")
				}
				calPath = sess.Calendars[0].Href
			}

			event := ical.Event{
				Summary:     summary,
				Start:       startT.UTC(),
				End:         endT.UTC(),
				Description: desc,
				Location:    loc,
			}
			icsData := ical.BuildEvent(event)
			parsed := ical.ParseEvents(icsData)
			uid := ""
			if len(parsed) > 0 {
				uid = parsed[0].UID
			}

			if err := dav.PutEvent(ctx, sess.Client, calPath, uid, icsData, ""); err != nil {
				return nil, fmt.Errorf("create event: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Event created.\nUID: %s\nCalendar: %s", uid, calPath),
				}},
			}, nil
		},
	)

	// calendar_create_recurring_event
	s.AddTool(
		"calendar_create_recurring_event",
		"Create a recurring calendar event with RRULE",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Event title"},
				"start":       {Type: "string", Description: "First occurrence start, ISO 8601"},
				"end":         {Type: "string", Description: "First occurrence end, ISO 8601"},
				"rrule":       {Type: "string", Description: "RFC 5545 RRULE, e.g. FREQ=WEEKLY;BYDAY=MO,WE,FR"},
				"description": {Type: "string", Description: "Event description (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"summary", "start", "end", "rrule"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary", "start", "end", "rrule"},
				Optional: []string{"description", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}
			return stub("calendar_create_recurring_event"), nil
		},
	)

	// calendar_update_event
	s.AddTool(
		"calendar_update_event",
		"Update an existing calendar event",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Event UID"},
				"summary":     {Type: "string", Description: "New title (optional)"},
				"start":       {Type: "string", Description: "New start, ISO 8601 (optional)"},
				"end":         {Type: "string", Description: "New end, ISO 8601 (optional)"},
				"description": {Type: "string", Description: "New description (optional)"},
				"location":    {Type: "string", Description: "New location (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"summary", "start", "end", "description", "location", "account"},
			}, args); err != nil {
				return nil, err
			}
			return stub("calendar_update_event"), nil
		},
	)

	// calendar_delete_event
	s.AddTool(
		"calendar_delete_event",
		"Delete a calendar event by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":     {Type: "string", Description: "Event UID"},
				"account": {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"account"},
			}, args); err != nil {
				return nil, err
			}
			return stub("calendar_delete_event"), nil
		},
	)
}

// formatCalendars renders the session state as human-readable text for the LLM.
func formatCalendars(accountName string, s *dav.Session) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Account: %s\n", accountName)
	fmt.Fprintf(&b, "Connected to %s\n", s.Client.BaseURL)
	fmt.Fprintf(&b, "Calendar home: %s\n", s.CalendarHome)
	if s.AddressbookHome != "" {
		fmt.Fprintf(&b, "Addressbook home: %s\n", s.AddressbookHome)
	}
	if len(s.Calendars) == 0 {
		fmt.Fprintf(&b, "No calendars found.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "Calendars (%d):\n", len(s.Calendars))
	for _, c := range s.Calendars {
		name := c.DisplayName
		if name == "" {
			name = "(no name)"
		}
		comps := ""
		if len(c.Components) > 0 {
			comps = fmt.Sprintf(" [%s]", strings.Join(c.Components, ","))
		}
		fmt.Fprintf(&b, "  - %s  %s%s\n", name, c.Href, comps)
	}
	return b.String()
}

// formatEvents renders a list of parsed events as human-readable text.
func formatEvents(events []ical.ParsedEvent, start, end time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Events from %s to %s (%d found):\n",
		start.Format(time.RFC3339), end.Format(time.RFC3339), len(events))
	for _, e := range events {
		fmt.Fprintf(&b, "\n[%s]\n", e.UID)
		fmt.Fprintf(&b, "  Summary: %s\n", e.Summary)
		fmt.Fprintf(&b, "  Start:   %s\n", e.Start.Format(time.RFC3339))
		fmt.Fprintf(&b, "  End:     %s\n", e.End.Format(time.RFC3339))
		if e.Location != "" {
			fmt.Fprintf(&b, "  Location: %s\n", e.Location)
		}
		if e.Description != "" {
			fmt.Fprintf(&b, "  Description: %s\n", e.Description)
		}
	}
	if len(events) == 0 {
		b.WriteString("No events found in range.\n")
	}
	return b.String()
}
