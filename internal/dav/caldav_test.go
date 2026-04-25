package dav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const calendarQueryResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/calendars/user/personal/ev1.ics</href>
    <propstat>
      <prop>
        <getetag>"etag1"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VEVENT
UID:ev1@test
SUMMARY:Meeting
DTSTART:20260501T100000Z
DTEND:20260501T110000Z
END:VEVENT
END:VCALENDAR
</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/calendars/user/personal/ev2.ics</href>
    <propstat>
      <prop>
        <getetag>"etag2"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VEVENT
UID:ev2@test
SUMMARY:Lunch
DTSTART:20260502T120000Z
DTEND:20260502T130000Z
END:VEVENT
END:VCALENDAR
</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestQueryEvents_ReturnsTwoEvents(t *testing.T) {
	var gotMethod, gotPath, gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(calendarQueryResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	calendars, err := QueryEvents(context.Background(), c, "/calendars/user/personal/",
		"20260501T000000Z", "20260531T235959Z")
	if err != nil {
		t.Fatalf("QueryEvents: %v", err)
	}
	if gotMethod != "REPORT" {
		t.Errorf("method=%q, want REPORT", gotMethod)
	}
	if gotPath != "/calendars/user/personal/" {
		t.Errorf("path=%q", gotPath)
	}
	if !strings.Contains(gotBody, "20260501T000000Z") {
		t.Errorf("request body missing start time")
	}
	if len(calendars) != 2 {
		t.Errorf("expected 2 calendar-data blobs, got %d", len(calendars))
	}
}

func TestQueryEvents_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	events, err := QueryEvents(context.Background(), c, "/cal/", "20260101T000000Z", "20260131T235959Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0, got %d", len(events))
	}
}

func TestPutEvent_SendsPUT(t *testing.T) {
	var gotMethod, gotPath, gotCT, gotINM string
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotINM = r.Header.Get("If-None-Match")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(201)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	icsData := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:abc@test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	err := PutEvent(context.Background(), c, "/calendars/user/personal/", "abc@test", icsData, "")
	if err != nil {
		t.Fatalf("PutEvent: %v", err)
	}
	if gotMethod != "PUT" {
		t.Errorf("method=%q, want PUT", gotMethod)
	}
	if gotPath != "/calendars/user/personal/abc@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
	if !strings.HasPrefix(gotCT, "text/calendar") {
		t.Errorf("Content-Type=%q", gotCT)
	}
	if gotINM != "*" {
		t.Errorf("If-None-Match=%q, want *", gotINM)
	}
	if !strings.Contains(gotBody, "UID:abc@test") {
		t.Errorf("body missing UID")
	}
}

func TestDeleteEvent_SendsDELETE(t *testing.T) {
	var gotMethod, gotPath, gotIfMatch string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotIfMatch = r.Header.Get("If-Match")
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	err := DeleteEvent(context.Background(), c, "/calendars/user/personal/", "abc@test", `"etag-abc"`)
	if err != nil {
		t.Fatalf("DeleteEvent: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method=%q, want DELETE", gotMethod)
	}
	if gotPath != "/calendars/user/personal/abc@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
	if gotIfMatch != `"etag-abc"` {
		t.Errorf("If-Match=%q, want \"etag-abc\"", gotIfMatch)
	}
}
