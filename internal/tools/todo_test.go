package tools_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/ical"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

// connectTodo creates a VTODO-capable CalDAV test session and returns cfg + cleanup.
func connectTodo(t *testing.T, extraHandler http.HandlerFunc) (config.Config, func()) {
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
      <c:supported-calendar-component-set><c:comp name="VTODO"/></c:supported-calendar-component-set>
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
			Name:     "todo-test",
			URL:      srv.URL,
			Username: "user",
			Password: "pass",
		}},
	}
	if _, err := dav.Connect(context.Background(), "todo-test", srv.URL, "user", "pass"); err != nil {
		srv.Close()
		t.Fatalf("dav.Connect: %v", err)
	}
	return cfg, srv.Close
}

func todoServer(t *testing.T, cfg config.Config) *mcp.Server {
	t.Helper()
	s := mcp.NewServer("test", "0")
	tools.RegisterTodo(s, cfg)
	return s
}

// todoVCalendar returns a minimal VTODO REPORT response for the given uid/summary/status.
func todoVCalendar(uid, summary, status string) string {
	data := ical.BuildTodo(ical.Todo{
		UID:     uid,
		Summary: summary,
		Status:  status,
	})
	return fmt.Sprintf(`<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/calendars/user/personal/%s.ics</href>
    <propstat><prop>
      <getetag>"etag-todo-1"</getetag>
      <c:calendar-data>%s</c:calendar-data>
    </prop><status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`, uid, data)
}

// ---- calendar_todo_list -------------------------------------------------------

func TestTodoList_Empty(t *testing.T) {
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_list", map[string]any{
		"account": "todo-test",
	})
	if err != nil {
		t.Fatalf("calendar_todo_list: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !strings.Contains(toolText(t, res), "No todos") {
		t.Errorf("expected 'No todos', got: %s", toolText(t, res))
	}
}

func TestTodoList_WithStatus(t *testing.T) {
	uid := "vtodo-status-test"
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(todoVCalendar(uid, "Write tests", "NEEDS-ACTION")))
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_list", map[string]any{
		"account": "todo-test",
		"status":  "NEEDS-ACTION",
	})
	if err != nil {
		t.Fatalf("calendar_todo_list: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !strings.Contains(toolText(t, res), "Write tests") {
		t.Errorf("expected todo summary in output, got: %s", toolText(t, res))
	}
}

func TestTodoList_StatusFilter_Excludes(t *testing.T) {
	uid := "vtodo-filter-excl"
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(todoVCalendar(uid, "Finished task", "COMPLETED")))
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_list", map[string]any{
		"account": "todo-test",
		"status":  "NEEDS-ACTION",
	})
	if err != nil {
		t.Fatalf("calendar_todo_list: %v", err)
	}
	if strings.Contains(toolText(t, res), "Finished task") {
		t.Error("COMPLETED todo should be excluded when filtering for NEEDS-ACTION")
	}
}

// ---- calendar_todo_create ----------------------------------------------------

func TestTodoCreate(t *testing.T) {
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_create", map[string]any{
		"summary": "Buy groceries",
		"account": "todo-test",
	})
	if err != nil {
		t.Fatalf("calendar_todo_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called")
	}
	if !strings.Contains(toolText(t, res), "Todo created") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

func TestTodoCreate_WithDue(t *testing.T) {
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_create", map[string]any{
		"summary": "Submit report",
		"due":     time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339),
		"account": "todo-test",
	})
	if err != nil {
		t.Fatalf("calendar_todo_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called")
	}
}

func TestTodoCreate_MissingSummary(t *testing.T) {
	cfg, cleanup := connectTodo(t, nil)
	defer cleanup()

	s := todoServer(t, cfg)
	_, err := s.CallTool(context.Background(), "calendar_todo_create", map[string]any{
		"account": "todo-test",
	})
	if err == nil {
		t.Fatal("expected error for missing summary")
	}
}

// ---- calendar_todo_update ----------------------------------------------------

func TestTodoUpdate(t *testing.T) {
	uid := "vtodo-upd-1"
	var putCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.Method {
		case "REPORT":
			w.WriteHeader(207)
			w.Write([]byte(todoVCalendar(uid, "Old summary", "NEEDS-ACTION")))
		case "PUT":
			putCalled = true
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_update", map[string]any{
		"uid":     uid,
		"summary": "New summary",
		"account": "todo-test",
	})
	if err != nil {
		t.Fatalf("calendar_todo_update: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT to be called")
	}
	if !strings.Contains(toolText(t, res), "Todo updated") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

// TestTodoUpdate_NotFound verifies that updating a non-existent UID returns an
// error (the tool surfaces it as a hard error via CallTool).
func TestTodoUpdate_NotFound(t *testing.T) {
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	_, err := s.CallTool(context.Background(), "calendar_todo_update", map[string]any{
		"uid":     "nonexistent-uid",
		"summary": "Irrelevant",
		"account": "todo-test",
	})
	if err == nil {
		t.Fatal("expected error for not-found uid")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// ---- calendar_todo_delete ----------------------------------------------------

func TestTodoDelete(t *testing.T) {
	uid := "vtodo-del-1"
	var deleteCalled bool
	extra := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.Method {
		case "REPORT":
			w.WriteHeader(207)
			w.Write([]byte(todoVCalendar(uid, "Task to delete", "NEEDS-ACTION")))
		case "DELETE":
			deleteCalled = true
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	})
	cfg, cleanup := connectTodo(t, extra)
	defer cleanup()

	s := todoServer(t, cfg)
	res, err := s.CallTool(context.Background(), "calendar_todo_delete", map[string]any{
		"uid":     uid,
		"account": "todo-test",
	})
	if err != nil {
		t.Fatalf("calendar_todo_delete: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !deleteCalled {
		t.Error("expected DELETE to be called")
	}
	if !strings.Contains(toolText(t, res), "Deleted todo") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

func TestTodoDelete_MissingUID(t *testing.T) {
	cfg, cleanup := connectTodo(t, nil)
	defer cleanup()

	s := todoServer(t, cfg)
	_, err := s.CallTool(context.Background(), "calendar_todo_delete", map[string]any{
		"account": "todo-test",
	})
	if err == nil {
		t.Fatal("expected error for missing uid")
	}
}
