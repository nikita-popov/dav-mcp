package tools

import (
	"context"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

func RegisterCalendar(s *mcp.Server, cfg config.Config) {

	// calendar_connect
	s.AddTool(
		"calendar_connect",
		"Connect to a CalDAV server and discover calendars",
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
			// TODO: PROPFIND /.well-known/caldav, discover principal and calendars
			return stub("calendar_connect"), nil
		},
	)

	// calendar_reconnect
	s.AddTool(
		"calendar_reconnect",
		"Reconnect to the CalDAV server using existing credentials from environment",
		mcp.InputSchema{Type: "object"},
		func(ctx context.Context, args map[string]any) (any, error) {
			// TODO: re-use cfg.DAVURL, cfg.Username, cfg.Password
			_ = cfg
			return stub("calendar_reconnect"), nil
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
			// TODO: CalDAV REPORT with time-range filter
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
			// TODO: generate UID, build iCalendar VEVENT, PUT to server
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
			// TODO: build VEVENT with RRULE property, PUT to server
			return stub("calendar_create_recurring_event"), nil
		},
	)

	// calendar_delete_event
	s.AddTool(
		"calendar_delete_event",
		"Delete a calendar event by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Event UID"},
				"calendar": {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: PROPFIND to resolve href by UID, then DELETE
			return stub("calendar_delete_event"), nil
		},
	)

	// calendar_get_todos
	s.AddTool(
		"calendar_get_todos",
		"List VTODO items from the calendar",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"completed": {Type: "boolean", Description: "Include completed todos (optional, default false)"},
				"calendar":  {Type: "string", Description: "Calendar path (optional)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Optional: []string{"completed", "calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: REPORT with comp-filter VTODO
			return stub("calendar_get_todos"), nil
		},
	)

	// calendar_create_todo
	s.AddTool(
		"calendar_create_todo",
		"Create a new VTODO task",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":  {Type: "string", Description: "Task title"},
				"due":      {Type: "string", Description: "Due date, ISO 8601 (optional)"},
				"priority": {Type: "integer", Description: "Priority 1 (high) to 9 (low), optional"},
				"notes":    {Type: "string", Description: "Additional notes (optional)"},
				"calendar": {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"summary"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary"},
				Optional: []string{"due", "priority", "notes", "calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: build VTODO, PUT to server
			return stub("calendar_create_todo"), nil
		},
	)

	// calendar_delete_todo
	s.AddTool(
		"calendar_delete_todo",
		"Delete a VTODO task by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Todo UID"},
				"calendar": {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: resolve href by UID, DELETE
			return stub("calendar_delete_todo"), nil
		},
	)

	// calendar_get_journals
	s.AddTool(
		"calendar_get_journals",
		"List VJOURNAL entries from the calendar",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"start":    {Type: "string", Description: "Range start, ISO 8601 (optional)"},
				"end":      {Type: "string", Description: "Range end, ISO 8601 (optional)"},
				"calendar": {Type: "string", Description: "Calendar path (optional)"},
			},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Optional: []string{"start", "end", "calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: REPORT with comp-filter VJOURNAL
			return stub("calendar_get_journals"), nil
		},
	)

	// calendar_get_journal
	s.AddTool(
		"calendar_get_journal",
		"Get a single VJOURNAL entry by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Journal UID"},
				"calendar": {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: REPORT with UID filter or GET by href
			return stub("calendar_get_journal"), nil
		},
	)

	// calendar_create_journal
	s.AddTool(
		"calendar_create_journal",
		"Create a new VJOURNAL entry",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Journal title"},
				"description": {Type: "string", Description: "Journal body text"},
				"dtstart":     {Type: "string", Description: "Entry date, ISO 8601 (optional, defaults to today)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"summary"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"summary"},
				Optional: []string{"description", "dtstart", "calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: build VJOURNAL, PUT to server
			return stub("calendar_create_journal"), nil
		},
	)

	// calendar_delete_journal
	s.AddTool(
		"calendar_delete_journal",
		"Delete a VJOURNAL entry by UID",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Journal UID"},
				"calendar": {Type: "string", Description: "Calendar path (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"calendar"},
			}, args); err != nil {
				return nil, err
			}
			// TODO: resolve href by UID, DELETE
			return stub("calendar_delete_journal"), nil
		},
	)
}
