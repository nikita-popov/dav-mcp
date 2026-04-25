package tools_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

// minimalCalDAVServer returns a test server that handles the three discovery
// steps and a catch-all for caldav-query / put / delete requests.
// extraRoutes are checked first; prefix "/calendars/user/personal" catches all
// sub-paths (uid.ics etc.) via the default switch arm.
func minimalCalDAVServer(t *testing.T, extraHandler http.HandlerFunc) *httptest.Server {
	t.Helper()

	const principalResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response><href>/</href>
    <propstat><prop><current-user-principal><href>/principals/user/</href></current-user-principal></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

	const calHomeResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response><href>/principals/user/</href>
    <propstat><prop><c:calendar-home-set><href>/calendars/user/</href></c:calendar-home-set></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

	const collectionsResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response><href>/calendars/user/personal/</href>
    <propstat><prop><displayname>Personal</displayname><resourcetype><collection/></resourcetype></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")

		// Extra handler intercepts all /calendars/user/personal/* paths.
		if extraHandler != nil && strings.HasPrefix(r.URL.Path, "/calendars/user/personal") {
			extraHandler(w, r)
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/calendars"):
			w.WriteHeader(207)
			w.Write([]byte(collectionsResp))
		case strings.HasPrefix(r.URL.Path, "/principals"):
			w.WriteHeader(207)
			w.Write([]byte(calHomeResp))
		default:
			w.WriteHeader(207)
			w.Write([]byte(principalResp))
		}
	}))
}

func calendarServer(t *testing.T, cfg config.Config) *mcp.Server {
	t.Helper()
	s := mcp.NewServer("test", "0")
	tools.RegisterCalendar(s, cfg)
	return s
}

// connectCalDAV spins up a CalDAV test server, calls dav.Connect and returns
// the config pointing at that server plus a cleanup function.
func connectCalDAV(t *testing.T, extraHandler http.HandlerFunc) (config.Config, func()) {
	t.Helper()
	srv := minimalCalDAVServer(t, extraHandler)
	cfg := config.Config{
		Accounts: []config.Account{{
			Name:     "default",
			URL:      srv.URL,
			Username: "user",
			Password: "pass",
		}},
	}
	_, err := dav.Connect(context.Background(), "default", srv.URL, "user", "pass")
	if err != nil {
		srv.Close()
		t.Fatalf("dav.Connect: %v", err)
	}
	return cfg, func() { srv.Close() }
}

// ---- calendar_list -------------------------------------------------

func TestCalendarCalendarList(t *testing.T) {
	cfg, cleanup := connectCalDAV(t, nil)
	defer cleanup()

	s := calendarServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_list", map[string]any{})
	if err != nil {
		t.Fatalf("calendar_list: %v", err)
	}
	text := toolText(t, res)
	if !strings.Contains(text, "Personal") {
		t.Errorf("expected calendar name in output, got: %s", text)
	}
}

// ---- calendar_event_list ----------------------------------------------------

func TestCalendarEventList_Empty(t *testing.T) {
	emptyReport := `<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "REPORT" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(207)
			w.Write([]byte(emptyReport))
			return
		}
		w.WriteHeader(405)
	})
	cfg, cleanup := connectCalDAV(t, extra)
	defer cleanup()

	s := calendarServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_event_list", map[string]any{
		"start": "2026-04-01T00:00:00Z",
		"end":   "2026-04-30T23:59:59Z",
	})
	if err != nil {
		t.Fatalf("calendar_event_list: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("unexpected tool error: %s", toolText(t, res))
	}
	text := toolText(t, res)
	if !strings.Contains(text, "No events") {
		t.Errorf("expected 'No events' in output, got: %s", text)
	}
}

func TestCalendarEventList_MissingParams(t *testing.T) {
	cfg, cleanup := connectCalDAV(t, nil)
	defer cleanup()

	s := calendarServer(t, cfg)
	_, err := s.CallTool(context.Background(), "calendar_event_list", map[string]any{
		"start": "2026-04-01T00:00:00Z",
		// end missing
	})
	if err == nil {
		t.Fatal("expected error for missing 'end' param")
	}
}

// ---- calendar_event_create --------------------------------------------------

func TestCalendarEventCreate(t *testing.T) {
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		// PROPFIND for collection listing during connect — respond normally.
		w.WriteHeader(405)
	})
	cfg, cleanup := connectCalDAV(t, extra)
	defer cleanup()

	s := calendarServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_event_create", map[string]any{
		"summary": "Test Event",
		"start":   "2026-05-01T10:00:00Z",
		"end":     "2026-05-01T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("calendar_event_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called on server")
	}
	if !strings.Contains(toolText(t, res), "Event created") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

// ---- calendar_event_recurring_create ----------------------------------------

func TestCalendarEventCreateRecurring(t *testing.T) {
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
	})
	cfg, cleanup := connectCalDAV(t, extra)
	defer cleanup()

	s := calendarServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_event_recurring_create", map[string]any{
		"summary": "Weekly standup",
		"start":   "2026-05-04T09:00:00Z",
		"end":     "2026-05-04T09:30:00Z",
		"rrule":   "FREQ=WEEKLY;BYDAY=MO",
	})
	if err != nil {
		t.Fatalf("calendar_event_recurring_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called on server")
	}
	if !strings.Contains(toolText(t, res), "FREQ=WEEKLY") {
		t.Errorf("expected RRULE in output, got: %s", toolText(t, res))
	}
}

// ---- helpers ----------------------------------------------------------------

func toolText(t *testing.T, res any) string {
	t.Helper()
	tr, ok := res.(mcp.ToolResult)
	if !ok || len(tr.Content) == 0 {
		return ""
	}
	return tr.Content[0].Text
}

func toolIsError(res any) bool {
	tr, ok := res.(mcp.ToolResult)
	return ok && tr.IsError
}
