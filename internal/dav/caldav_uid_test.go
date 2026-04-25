package dav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const eventByUIDResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/calendars/user/personal/ev1.ics</href>
    <propstat>
      <prop>
        <getetag>"etag-uid1"</getetag>
        <c:calendar-data>BEGIN:VCALENDAR
BEGIN:VEVENT
UID:uid1@test
SUMMARY:Sprint review
DTSTART:20260601T090000Z
DTEND:20260601T100000Z
END:VEVENT
END:VCALENDAR
</c:calendar-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestQueryEventByUID_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(eventByUIDResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	rec, err := QueryEventByUID(context.Background(), c, "/calendars/user/personal/", "uid1@test")
	if err != nil {
		t.Fatalf("QueryEventByUID: %v", err)
	}
	if rec.Event.UID != "uid1@test" {
		t.Errorf("UID=%q", rec.Event.UID)
	}
	if rec.Href != "/calendars/user/personal/ev1.ics" {
		t.Errorf("Href=%q", rec.Href)
	}
	if rec.ETag != "\"etag-uid1\"" {
		t.Errorf("ETag=%q", rec.ETag)
	}
}

func TestQueryEventByUID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	_, err := QueryEventByUID(context.Background(), c, "/calendars/user/personal/", "missing@test")
	if err == nil {
		t.Fatal("expected error for missing UID")
	}
}

func TestPutEventHref_SendsPUT(t *testing.T) {
	var gotPath, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/calendars") {
			gotPath = r.URL.Path
			gotCT = r.Header.Get("Content-Type")
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	err := PutEventHref(context.Background(), c, "/calendars/user/personal/ev1.ics", "BEGIN:VCALENDAR\r\nEND:VCALENDAR\r\n", "\"etag-uid1\"")
	if err != nil {
		t.Fatalf("PutEventHref: %v", err)
	}
	if gotPath != "/calendars/user/personal/ev1.ics" {
		t.Errorf("path=%q", gotPath)
	}
	if !strings.HasPrefix(gotCT, "text/calendar") {
		t.Errorf("Content-Type=%q", gotCT)
	}
}
