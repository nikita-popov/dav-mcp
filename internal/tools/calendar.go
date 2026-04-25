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

	// calendar_list
	s.AddTool(
		"calendar_list",
		"List all calendars across connected accounts. Call this first to discover available calendars and their paths before using calendar_event_list or calendar_event_create.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"account": {Type: "string", Description: "Account name (optional, lists all accounts if omitted)"},
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

			var b strings.Builder
			for _, acc := range accounts {
				sess := dav.Get(acc.Name)
				if sess == nil {
					fmt.Fprintf(&b, "Account %q: not connected (use dav_reconnect)\n", acc.Name)
					continue
				}
				b.WriteString(formatCalendars(acc.Name, sess))
			}
			if b.Len() == 0 {
				return nil, fmt.Errorf("no accounts connected; use dav_reconnect first")
			}
			return mcp.ToolResult{
				Content: []mcp.ContentItem{{Type: "text", Text: b.String()}},
			}, nil
		},
	)

	// dav_connect
	s.AddTool(
		"dav_connect",
		"Connect to a CalDAV/CardDAV server and discover calendars and address books. Returns a list of available calendars.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"url":      {Type: "string", Description: "DAV server URL"},
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

	// dav_reconnect
	s.AddTool(
		"dav_reconnect",
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

	// calendar_event_list
	s.AddTool(
		"calendar_event_list",
		"List calendar events in a time range.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"start":    {Type: "string", Description: "Range start, ISO 8601, e.g. 2026-04-01T00:00:00Z"},
				"end":      {Type: "string", Description: "Range end, ISO 8601, e.g. 2026-04-30T23:59:59Z"},
				"calendar": {Type: "string", Description: "Calendar path from calendar_list (optional, defaults to primary calendar of the account)"},
				"account":  {Type: "string", Description: "Account name (optional)"},
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

	// calendar_event_create
	s.AddTool(
		"calendar_event_create",
		"Create a new calendar event. Returns the UID of the created event.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Event title"},
				"start":       {Type: "string", Description: "Start datetime, ISO 8601, e.g. 2026-05-01T10:00:00Z"},
				"end":         {Type: "string", Description: "End datetime, ISO 8601"},
				"description": {Type: "string", Description: "Event description (optional)"},
				"location":    {Type: "string", Description: "Location (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path from calendar_list (optional)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
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

	// calendar_event_recurring_create
	s.AddTool(
		"calendar_event_recurring_create",
		"Create a recurring calendar event with RRULE",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"summary":     {Type: "string", Description: "Event title"},
				"start":       {Type: "string", Description: "First occurrence start, ISO 8601"},
				"end":         {Type: "string", Description: "First occurrence end, ISO 8601"},
				"rrule":       {Type: "string", Description: "RFC 5545 RRULE, e.g. FREQ=WEEKLY;BYDAY=MO,WE,FR"},
				"description": {Type: "string", Description: "Event description (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path from calendar_list (optional)"},
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

			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			summary, _ := args["summary"].(string)
			startStr, _ := args["start"].(string)
			endStr, _ := args["end"].(string)
			rrule, _ := args["rrule"].(string)
			desc, _ := args["description"].(string)

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
				RRule:       rrule,
			}
			icsData := ical.BuildEvent(event)
			parsed := ical.ParseEvents(icsData)
			uid := ""
			if len(parsed) > 0 {
				uid = parsed[0].UID
			}

			if err := dav.PutEvent(ctx, sess.Client, calPath, uid, icsData, ""); err != nil {
				return nil, fmt.Errorf("create recurring event: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Recurring event created.\nUID: %s\nRRULE: %s\nCalendar: %s", uid, rrule, calPath),
				}},
			}, nil
		},
	)

	// calendar_event_update
	s.AddTool(
		"calendar_event_update",
		"Update an existing calendar event by UID. Only the fields you provide are changed.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":         {Type: "string", Description: "Event UID (required)"},
				"summary":     {Type: "string", Description: "New title (optional)"},
				"start":       {Type: "string", Description: "New start, ISO 8601 (optional)"},
				"end":         {Type: "string", Description: "New end, ISO 8601 (optional)"},
				"description": {Type: "string", Description: "New description (optional)"},
				"location":    {Type: "string", Description: "New location (optional)"},
				"calendar":    {Type: "string", Description: "Calendar path (optional, searches all calendars if omitted)"},
				"account":     {Type: "string", Description: "Account name (optional)"},
			},
			Required: []string{"uid"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			if err := mcp.ValidateArgs(mcp.ArgSchema{
				Required: []string{"uid"},
				Optional: []string{"summary", "start", "end", "description", "location", "calendar", "account"},
			}, args); err != nil {
				return nil, err
			}

			sess, err := session(ctx, cfg, strArg(args, "account"))
			if err != nil {
				return nil, err
			}

			uid := strArg(args, "uid")
			calPath := strArg(args, "calendar")

			ref, err := findEventByUID(ctx, sess, uid, calPath)
			if err != nil {
				return nil, err
			}

			// Convert ParsedEvent to Event, then patch only supplied fields.
			ev := parsedToEvent(ref.rec.Event)
			ev.Sequence++
			if v := strArg(args, "summary"); v != "" {
				ev.Summary = v
			}
			if v := strArg(args, "start"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return nil, fmt.Errorf("invalid start: %w", err)
				}
				ev.Start = t.UTC()
			}
			if v := strArg(args, "end"); v != "" {
				t, err := time.Parse(time.RFC3339, v)
				if err != nil {
					return nil, fmt.Errorf("invalid end: %w", err)
				}
				ev.End = t.UTC()
			}
			if _, ok := args["description"]; ok {
				ev.Description = strArg(args, "description")
			}
			if _, ok := args["location"]; ok {
				ev.Location = strArg(args, "location")
			}

			icsData := ical.BuildEvent(ev)
			if err := dav.PutEventHref(ctx, sess.Client, ref.rec.Href, icsData, ref.rec.ETag); err != nil {
				return nil, fmt.Errorf("calendar_event_update: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Event updated.\nUID: %s", uid),
				}},
			}, nil
		},
	)

	// calendar_event_delete
	s.AddTool(
		"calendar_event_delete",
		"Delete a calendar event by UID.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"uid":      {Type: "string", Description: "Event UID"},
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

			uid := strArg(args, "uid")
			calPath := strArg(args, "calendar")

			ref, err := findEventByUID(ctx, sess, uid, calPath)
			if err != nil {
				return nil, fmt.Errorf("calendar_event_delete: %w", err)
			}

			if err := sess.Client.Delete(ctx, ref.rec.Href, ref.rec.ETag); err != nil {
				return nil, fmt.Errorf("calendar_event_delete: %w", err)
			}

			return mcp.ToolResult{
				Content: []mcp.ContentItem{{
					Type: "text",
					Text: fmt.Sprintf("Deleted event UID=%s from %s", uid, ref.calendarHref),
				}},
			}, nil
		},
	)
}

// ---- helpers ----------------------------------------------------------------

// eventRef pairs an EventRecord with the calendar collection href it came from.
type eventRef struct {
	rec          *dav.EventRecord
	calendarHref string
}

// findEventByUID searches calendars in the session for an event with the given UID.
// If calendarHref is non-empty, only that collection is searched.
func findEventByUID(ctx context.Context, sess *dav.Session, uid, calendarHref string) (*eventRef, error) {
	calendars := sess.Calendars
	if calendarHref != "" {
		calendars = []dav.Collection{{Href: calendarHref}}
	}
	for _, cal := range calendars {
		rec, err := dav.QueryEventByUID(ctx, sess.Client, cal.Href, uid)
		if err != nil {
			// not found in this calendar — keep searching
			continue
		}
		return &eventRef{rec: rec, calendarHref: cal.Href}, nil
	}
	return nil, fmt.Errorf("event UID=%q not found", uid)
}

// parsedToEvent converts a ParsedEvent (read from server) to an Event (for building iCal).
func parsedToEvent(p ical.ParsedEvent) ical.Event {
	return ical.Event{
		UID:         p.UID,
		Summary:     p.Summary,
		Description: p.Description,
		Location:    p.Location,
		Start:       p.Start,
		End:         p.End,
		RRule:       p.RRule,
		Sequence:    p.Sequence,
	}
}

func formatCalendars(accName string, sess *dav.Session) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Account: %s\n", accName)
	fmt.Fprintf(&b, "Calendar home: %s\n", sess.CalendarHome)
	if sess.AddressbookHome != "" {
		fmt.Fprintf(&b, "Addressbook home: %s\n", sess.AddressbookHome)
	}
	for _, cal := range sess.Calendars {
		fmt.Fprintf(&b, "  - %s (%s)\n", cal.DisplayName, cal.Href)
	}
	return b.String()
}

func formatEvents(events []ical.ParsedEvent, start, end time.Time) string {
	if len(events) == 0 {
		return fmt.Sprintf("No events found between %s and %s.", start.Format(time.RFC3339), end.Format(time.RFC3339))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d event(s) between %s and %s:\n\n",
		len(events), start.Format(time.RFC3339), end.Format(time.RFC3339))
	for _, ev := range events {
		fmt.Fprintf(&b, "UID: %s\n", ev.UID)
		fmt.Fprintf(&b, "Summary: %s\n", ev.Summary)
		fmt.Fprintf(&b, "Start: %s\n", ev.Start.Format(time.RFC3339))
		fmt.Fprintf(&b, "End: %s\n", ev.End.Format(time.RFC3339))
		if ev.Description != "" {
			fmt.Fprintf(&b, "Description: %s\n", ev.Description)
		}
		if ev.Location != "" {
			fmt.Fprintf(&b, "Location: %s\n", ev.Location)
		}
		b.WriteString("\n")
	}
	return b.String()
}
