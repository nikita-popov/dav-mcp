package dav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientGet_ReturnsBody(t *testing.T) {
	const body = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:x@test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method=%q, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "text/calendar")
		w.Header().Set("ETag", "\"etag-x\"")
		w.WriteHeader(200)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	got, etag, err := c.Get(context.Background(), "/calendars/user/personal/x@test.ics")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(string(got), "UID:x@test") {
		t.Errorf("body missing UID, got: %s", string(got))
	}
	if etag != "\"etag-x\"" {
		t.Errorf("etag=%q", etag)
	}
}

func TestClientGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	_, _, err := c.Get(context.Background(), "/calendars/user/personal/nope.ics")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
