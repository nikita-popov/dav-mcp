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
		"List calendar events in a time range across all accounts. Returns one compact line per event. Use calendar_event_get for full details of a specific event.",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"start":    {Type: "string", Description: "Range start, ISO 8601, e.g. 2026-04-01T00:00:00Z"},
				"end":      {Type: "string", Description: "Range end, ISO 8601, e.g. 2026-04-30T23:59:59Z"},
				"calendar": {Type: "string", Description: "Calendar path from calendar_list (optional; queries all calendars of all accounts if omitted)"},
				"account":  {Type: "string", Description: "Account name (optional; queries all accounts if omitted)"},
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

			accName := strArg(args, "account")
			calPath := strArg(args, "calendar")

			const icalFmt = "20060102T150405Z"
			start := startT.UTC().Format(icalFmt)
			end := endT.UTC().Format(icalFmt)

			var allEvents []ical.ParsedEvent

			if accName != "" {
				// Single account, explicit.
				sess, err := session(ctx, cfg, accName)
				if err != nil {
					return nil, err
				}
				path := calPath
				if path == "" {
					if len(sess.Calendars) == 0 {
						return nil, fmt.Errorf("no calendars found in account %q", accName)
					}
					path = sess.Calendars[0].Href
				}
				events, err := queryCalendarEvents(ctx, sess, path, start, end)
				if err != nil {
					return nil, err
				}
				allEvents = append(allEvents, events...)
			} else if calPath != "" {
				// Explicit calendar path but no account — try all sessions.
				for _, acc := range cfg.Accounts {
					sess, err := session(ctx, cfg, acc.Name)
					if err != nil {
						continue
					}
					events, err := queryCalendarEvents(ctx, sess, calPath, start, end)
					if err != nil {
						continue
					}
					allEvents = append(allEvents, events...)
				}
			} else {
				// No account, no calendar — query every calendar of every account.
				for _, acc := range cfg.Accounts {
					sess, err := session(ctx, cfg, acc.Name)
					if err != nil {
						mcp.Debugf("calendar_event_list: skip account=%q: %v", acc.Name, err)
						continue
					}
					for _, cal := range sess.Calendars {
						events, err := queryCalendarEvents(ctx, sess, cal.Href, start, end)
						if err != nil {
							mcp.Debugf("calendar_event_list: skip calendar=%q: %v", cal.Href, err)
							continue
						}
						allEvents = append(allEvents, events...)
					}
				}
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

type eventRef struct {
	rec          *dav.EventRecord
	calendarHref string
}

// queryCalendarEvents fetches and parses events from a single calendar collection.
func queryCalendarEvents(ctx context.Context, sess *dav.Session, calPath, start, end string) ([]ical.ParsedEvent, error) {
	blobs, err := dav.QueryEvents(ctx, sess.Client, calPath, start, end)
	if err != nil {
		return nil, err
	}
	var events []ical.ParsedEvent
	for _, blob := range blobs {
		events = append(events, ical.ParseEvents(blob)...)
	}
	return events, nil
}

func findEventByUID(ctx context.Context, sess *dav.Session, uid, calendarHref string) (*eventRef, error) {
	calendars := sess.Calendars
	if calendarHref != "" {
		calendars = []dav.Collection{{Href: calendarHref}}
	}
	for _, cal := range calendars {
		rec, err := dav.QueryEventByUID(ctx, sess.Client, cal.Href, uid)
		if err != nil {
			continue
		}
		return &eventRef{rec: rec, calendarHref: cal.Href}, nil
	}
	return nil, fmt.Errorf("event UID=%q not found", uid)
}

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

// formatEvents renders a compact one-line-per-event list.
// Format: MM-DD HH:MMtz–HH:MM [rec] Summary  uid:<uid>
func formatEvents(events []ical.ParsedEvent, start, end time.Time) string {
	if len(events) == 0 {
		return fmt.Sprintf("No events found between %s and %s.",
			start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d event(s) %s – %s:\n",
		len(events), start.Format("2006-01-02"), end.Format("2006-01-02"))
	for _, ev := range events {
		startFmt := ev.Start.Format("01-02 15:04Z07:00")
		endFmt := ev.End.Format("15:04Z07:00")
		if ev.StartTZ != "" {
			if loc, err := time.LoadLocation(ev.StartTZ); err == nil {
				startFmt = ev.Start.In(loc).Format("01-02 15:04Z07:00")
				endFmt = ev.End.In(loc).Format("15:04Z07:00")
			}
		}
		rec := ""
		if ev.RRule != "" {
			rec = " [rec]"
		}
		fmt.Fprintf(&b, "%s–%s%s %s  uid:%s\n",
			startFmt, endFmt, rec, ev.Summary, ev.UID)
	}
	return b.String()
}
