package dav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const journalQueryResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/calendars/user/journal/j1.ics</href>
    <propstat>
      <prop>
        <getetag>"etag1"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VJOURNAL
UID:j1@test
SUMMARY:Stand-up notes
DTSTART:20260501
END:VJOURNAL
END:VCALENDAR
</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestQueryJournals_ReturnsOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Errorf("method=%q, want REPORT", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), "VJOURNAL") {
			t.Errorf("body missing VJOURNAL filter")
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(journalQueryResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	journals, err := QueryJournals(context.Background(), c, "/calendars/user/journal/")
	if err != nil {
		t.Fatalf("QueryJournals: %v", err)
	}
	if len(journals) != 1 {
		t.Errorf("expected 1, got %d", len(journals))
	}
}

func TestQueryJournalByUID_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(journalQueryResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	rec, err := QueryJournalByUID(context.Background(), c, "/calendars/user/journal/", "j1@test")
	if err != nil {
		t.Fatalf("QueryJournalByUID: %v", err)
	}
	if rec.Journal.UID != "j1@test" {
		t.Errorf("UID=%q", rec.Journal.UID)
	}
	if rec.Href != "/calendars/user/journal/j1.ics" {
		t.Errorf("Href=%q", rec.Href)
	}
}

func TestQueryJournalByUID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	_, err := QueryJournalByUID(context.Background(), c, "/calendars/user/journal/", "missing@test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPutJournal_SendsPUT(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/calendars/user/journal") {
			gotMethod = r.Method
			gotPath = r.URL.Path
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	icsData := "BEGIN:VCALENDAR\r\nBEGIN:VJOURNAL\r\nUID:j1@test\r\nEND:VJOURNAL\r\nEND:VCALENDAR\r\n"
	err := PutJournal(context.Background(), c, "/calendars/user/journal/", "j1@test", icsData, "")
	if err != nil {
		t.Fatalf("PutJournal: %v", err)
	}
	if gotMethod != "PUT" {
		t.Errorf("method=%q, want PUT", gotMethod)
	}
	if gotPath != "/calendars/user/journal/j1@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
}

func TestPutJournalHref_SendsPUT(t *testing.T) {
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
	err := PutJournalHref(context.Background(), c, "/calendars/user/journal/j1@test.ics", "BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n", "\"etag1\"")
	if err != nil {
		t.Fatalf("PutJournalHref: %v", err)
	}
	if gotPath != "/calendars/user/journal/j1@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
}

func TestDeleteJournal_SendsDELETE(t *testing.T) {
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
	err := DeleteJournal(context.Background(), c, "/calendars/user/journal/j1@test.ics", "")
	if err != nil {
		t.Fatalf("DeleteJournal: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method=%q, want DELETE", gotMethod)
	}
	if gotPath != "/calendars/user/journal/j1@test.ics" {
		t.Errorf("path=%q", gotPath)
	}
}
