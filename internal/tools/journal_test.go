package tools_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/ical"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

// connectJournal creates a VJOURNAL-capable CalDAV test session and returns cfg + cleanup.
func connectJournal(t *testing.T, extraHandler http.HandlerFunc) (config.Config, func()) {
	t.Helper()

	principalBody := []byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response><href>/</href>
    <propstat><prop><current-user-principal><href>/principals/user/</href></current-user-principal></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`)
	calHomeBody := []byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response><href>/principals/user/</href>
    <propstat><prop><c:calendar-home-set><href>/calendars/user/</href></c:calendar-home-set></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`)
	collectionsBody := []byte(`<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response><href>/calendars/user/personal/</href>
    <propstat><prop>
      <displayname>Personal</displayname>
      <resourcetype><collection/></resourcetype>
      <c:supported-calendar-component-set><c:comp name="VJOURNAL"/></c:supported-calendar-component-set>
    </prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if extraHandler != nil && strings.HasPrefix(r.URL.Path, "/calendars/user/personal") {
			extraHandler(w, r)
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/calendars"):
			w.WriteHeader(207)
			w.Write(collectionsBody)
		case strings.HasPrefix(r.URL.Path, "/principals"):
			w.WriteHeader(207)
			w.Write(calHomeBody)
		default:
			w.WriteHeader(207)
			w.Write(principalBody)
		}
	}))

	cfg := config.Config{
		Accounts: []config.Account{{
			Name:     "journal-test",
			URL:      srv.URL,
			Username: "user",
			Password: "pass",
		}},
	}
	if _, err := dav.Connect(context.Background(), "journal-test", srv.URL, "user", "pass"); err != nil {
		srv.Close()
		t.Fatalf("dav.Connect: %v", err)
	}
	return cfg, srv.Close
}

func journalServer(t *testing.T, cfg config.Config) *mcp.Server {
	t.Helper()
	s := mcp.NewServer("test", "0")
	tools.RegisterJournal(s, cfg)
	return s
}

// journalVCalendar returns a minimal VJOURNAL REPORT response for the given uid/summary/status.
func journalVCalendar(uid, summary, status string) string {
	data := ical.BuildJournal(ical.Journal{
		UID:     uid,
		Summary: summary,
		Status:  status,
	})
	return fmt.Sprintf(`<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/calendars/user/personal/%s.ics</href>
    <propstat><prop>
      <getetag>"etag-j-1"</getetag>
      <c:calendar-data>%s</c:calendar-data>
    </prop><status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`, uid, data)
}

// ---- calendar_journal_list --------------------------------------------------

func TestJournalList_Empty(t *testing.T) {
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_list", map[string]any{
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("calendar_journal_list: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !strings.Contains(toolText(t, res), "No journal") {
		t.Errorf("expected 'No journal entries', got: %s", toolText(t, res))
	}
}

func TestJournalList_WithEntries(t *testing.T) {
	uid := "vjournal-list-1"
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(journalVCalendar(uid, "Daily standup notes", "FINAL")))
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_list", map[string]any{
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("calendar_journal_list: %v", err)
	}
	if !strings.Contains(toolText(t, res), "Daily standup notes") {
		t.Errorf("expected journal summary in output, got: %s", toolText(t, res))
	}
}

func TestJournalList_StatusFilter(t *testing.T) {
	uid := "vjournal-filter-1"
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(journalVCalendar(uid, "Draft entry", "DRAFT")))
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_list", map[string]any{
		"account": "journal-test",
		"status":  "FINAL",
	})
	if err != nil {
		t.Fatalf("calendar_journal_list: %v", err)
	}
	if strings.Contains(toolText(t, res), "Draft entry") {
		t.Error("DRAFT entry should be excluded when filtering for FINAL")
	}
}

// ---- calendar_journal_create ------------------------------------------------

func TestJournalCreate(t *testing.T) {
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_create", map[string]any{
		"summary": "Meeting notes",
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("calendar_journal_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called")
	}
	if !strings.Contains(toolText(t, res), "Journal created") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

func TestJournalCreate_WithDate(t *testing.T) {
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_create", map[string]any{
		"summary": "Retrospective",
		"date":    "2026-04-25",
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("calendar_journal_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called")
	}
}

func TestJournalCreate_InvalidDate(t *testing.T) {
	cfg, cleanup := connectJournal(t, nil)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_create", map[string]any{
		"summary": "Bad date",
		"date":    "not-a-date",
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if !toolIsError(res) {
		t.Errorf("expected tool error for invalid date, got: %s", toolText(t, res))
	}
}

func TestJournalCreate_MissingSummary(t *testing.T) {
	cfg, cleanup := connectJournal(t, nil)
	defer cleanup()

	s := journalServer(t, cfg)
	_, err := s.CallTool(context.Background(), "calendar_journal_create", map[string]any{
		"account": "journal-test",
	})
	if err == nil {
		t.Fatal("expected error for missing summary")
	}
}

// ---- calendar_journal_update ------------------------------------------------

func TestJournalUpdate(t *testing.T) {
	uid := "vjournal-upd-1"
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.Method {
		case "REPORT":
			w.WriteHeader(207)
			w.Write([]byte(journalVCalendar(uid, "Old title", "DRAFT")))
		case "PUT":
			putCalled = true
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_update", map[string]any{
		"uid":     uid,
		"summary": "New title",
		"status":  "FINAL",
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("calendar_journal_update: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called")
	}
	if !strings.Contains(toolText(t, res), "Journal updated") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

func TestJournalUpdate_NotFound(t *testing.T) {
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_update", map[string]any{
		"uid":     "ghost-uid",
		"summary": "Irrelevant",
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if !toolIsError(res) {
		t.Errorf("expected tool error for not-found uid, got: %s", toolText(t, res))
	}
}

// ---- calendar_journal_delete ------------------------------------------------

func TestJournalDelete(t *testing.T) {
	uid := "vjournal-del-1"
	var deleteCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.Method {
		case "REPORT":
			w.WriteHeader(207)
			w.Write([]byte(journalVCalendar(uid, "Entry to delete", "FINAL")))
		case "DELETE":
			deleteCalled = true
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	})
	cfg, cleanup := connectJournal(t, extra)
	defer cleanup()

	s := journalServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_journal_delete", map[string]any{
		"uid":     uid,
		"account": "journal-test",
	})
	if err != nil {
		t.Fatalf("calendar_journal_delete: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !deleteCalled {
		t.Error("expected DELETE to be called")
	}
	if !strings.Contains(toolText(t, res), "Deleted journal") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

func TestJournalDelete_MissingUID(t *testing.T) {
	cfg, cleanup := connectJournal(t, nil)
	defer cleanup()

	s := journalServer(t, cfg)
	_, err := s.CallTool(context.Background(), "calendar_journal_delete", map[string]any{
		"account": "journal-test",
	})
	if err == nil {
		t.Fatal("expected error for missing uid")
	}
}
