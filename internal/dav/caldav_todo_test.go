package dav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const todoQueryResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/calendars/user/tasks/todo1.ics</href>
    <propstat>
      <prop>
        <getetag>"etag1"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VTODO
UID:todo1@test
SUMMARY:Buy milk
STATUS:NEEDS-ACTION
END:VTODO
END:VCALENDAR
</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestQueryTodos_ReturnsOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Errorf("method=%q, want REPORT", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "VTODO") {
			t.Errorf("body missing VTODO filter")
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(todoQueryResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	todos, err := QueryTodos(context.Background(), c, "/calendars/user/tasks/")
	if err != nil {
		t.Fatalf("QueryTodos: %v", err)
	}
	if len(todos) != 1 {
		t.Errorf("expected 1, got %d", len(todos))
	}
}

func TestQueryTodoByUID_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(todoQueryResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	rec, err := QueryTodoByUID(context.Background(), c, "/calendars/user/tasks/", "todo1@test")
	if err != nil {
		t.Fatalf("QueryTodoByUID: %v", err)
	}
	if rec.Todo.UID != "todo1@test" {
		t.Errorf("UID=%q", rec.Todo.UID)
	}
	if rec.Href != "/calendars/user/tasks/todo1.ics" {
		t.Errorf("Href=%q", rec.Href)
	}
}

func TestQueryTodoByUID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	_, err := QueryTodoByUID(context.Background(), c, "/calendars/user/tasks/", "missing@test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPutTodo_SendsPUT(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/calendars/user/tasks") {
			gotMethod = r.Method
			gotPath = r.URL.Path
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	icsData := "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:todo1@test\r\nEND:VTODO\r\nEND:VCALENDAR\r\n"
	err := PutTodo(context.Background(), c, "/calendars/user/tasks/", "todo1@test", icsData, "")
	if err != nil {
		t.Fatalf("PutTodo: %v", err)
	}
	if gotMethod != "PUT" {
		t.Errorf("method=%q, want PUT", gotMethod)
	}
	if gotPath != "/calendars/user/tasks/todo1@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
}

func TestPutTodoHref_SendsPUT(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/calendars") {
			gotPath = r.URL.Path
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	err := PutTodoHref(context.Background(), c, "/calendars/user/tasks/todo1@test.ics", "BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n", "\"etag1\"")
	if err != nil {
		t.Fatalf("PutTodoHref: %v", err)
	}
	if gotPath != "/calendars/user/tasks/todo1@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
}

func TestDeleteTodo_SendsDELETE(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/calendars") {
			gotMethod = r.Method
			gotPath = r.URL.Path
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	err := DeleteTodo(context.Background(), c, "/calendars/user/tasks/todo1@test.ics", "")
	if err != nil {
		t.Fatalf("DeleteTodo: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method=%q, want DELETE", gotMethod)
	}
	if gotPath != "/calendars/user/tasks/todo1@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
}
