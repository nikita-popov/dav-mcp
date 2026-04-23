package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
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
			},
			Required: []string{"url", "username", "password"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"url", "username", "password"},
			}, args); err != nil {
				return nil, err
			}
			rawURL, _ := args["url"].(string)
			username, _ := args["username"].(string)
			password, _ := args["password"].(string)

			session, err := dav.Connect(ctx, rawURL, username, password)
			if err != nil {
				return nil, err
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatCalendars(session),
				}},
			}, nil
		},
	)

	// calendar_reconnect
	s.AddTool(
		"calendar_reconnect",
		"Reconnect to the CalDAV server using credentials from environment variables (DAV_URL, DAV_USERNAME, DAV_PASSWORD).",
		mcp.InputSchema{Type: "object"},
		func(ctx context.Context, args map[string]any) (any, error) {
			if cfg.DAVURL == "" {
				return nil, fmt.Errorf("DAV_URL is not set")
			}
			session, err := dav.Connect(ctx, cfg.DAVURL, cfg.Username, cfg.Password)
			if err != nil {
				return nil, err
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: formatCalendars(session),
				}},
			}, nil
		},
	)

	// calendar_get_events
	s.AddTool(
		"calendar_get_events",
		"List calendar events in a given time range",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"start":    {Type: "string", Description: "Range start, ISO 8601, e.g. 2026-04-01T00:00:00Z"},
				"end":      {Type: "string", Description: "Range end, ISO 8601, e.g. 2026-04-30T23:59:59Z"},
				"calendar": {Type: "string", Description: "Calendar path (optional, defaults to primary)"},
			},
			Required: []string{"start", "end"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"start", "end"},
				Optional: []string{"calendar"},
			}, args); err != nil {
				return nil, err
			}
			return stub("calendar_get_events"), nil
		},
	)

	// calendar_create_event
	s.AddTool(
		"calendar_create_event",
		"Create a new calendar event",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Event title"},
				"start":       {Type: "string", Description: "Start datetime, ISO 8601"},
				"end":         {Type: "string", Description: "End datetime, ISO 8601"},
				"description": {Type: "string", Description: "Event description (optional)"},
				"location":    {Type: "string", Description: "Location (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"summary", "start", "end"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary", "start", "end"},
				Optional: []string{"description", "location", "calendar"},
			}, args); err != nil {
				return nil, err
			}
			return stub("calendar_create_event"), nil
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
			},
			Required: []string{"summary", "start", "end", "rrule"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary", "start", "end", "rrule"},
				Optional: []string{"description", "calendar"},
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
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"summary", "start", "end", "description", "location"},
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
				"uid": {Type: "string", Description: "Event UID"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
			}, args); err != nil {
				return nil, err
			}
			return stub("calendar_delete_event"), nil
		},
	)
}

// formatCalendars renders the session state as human-readable text for the LLM.
func formatCalendars(s *dav.Session) string {
	var b strings.Builder
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
		fmt.Fprintf(&b, "  - %s  [%s]\n", name, c.Href)
	}
	return b.String()
}
