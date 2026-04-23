package dav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 207 Multi-Status with current-user-principal
const principalResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/</href>
    <propstat>
      <prop>
        <current-user-principal><href>/principals/user/</href></current-user-principal>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

// 207 with calendar-home-set
const calHomeResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <response>
    <href>/principals/user/</href>
    <propstat>
      <prop>
        <c:calendar-home-set><href>/calendars/user/</href></c:calendar-home-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

// 207 depth:1 collections
const collectionsResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/calendars/user/</href>
    <propstat>
      <prop>
        <displayname>Home</displayname>
        <resourcetype><collection/></resourcetype>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/calendars/user/personal/</href>
    <propstat>
      <prop>
        <displayname>Personal</displayname>
        <resourcetype><collection/></resourcetype>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/calendars/user/note.ics</href>
    <propstat>
      <prop>
        <displayname>note</displayname>
        <resourcetype/>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c, err := New(srv.URL, "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	return c, srv
}

func TestDiscoverPrincipal(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(principalResp))
	})
	defer srv.Close()

	href, err := DiscoverPrincipal(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	if href != "/principals/user/" {
		t.Fatalf("expected /principals/user/, got %q", href)
	}
}

func TestDiscoverCalendarHome(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(calHomeResp))
	})
	defer srv.Close()

	href, err := DiscoverCalendarHome(context.Background(), c, "/principals/user/")
	if err != nil {
		t.Fatal(err)
	}
	if href != "/calendars/user/" {
		t.Fatalf("expected /calendars/user/, got %q", href)
	}
}

func TestDiscoverCollections(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(collectionsResp))
	})
	defer srv.Close()

	cols, err := DiscoverCollections(context.Background(), c, "/calendars/user/")
	if err != nil {
		t.Fatal(err)
	}
	// root skipped, note.ics (non-collection) skipped — only /calendars/user/personal/ remains
	if len(cols) != 1 {
		t.Fatalf("expected 1 collection, got %d: %+v", len(cols), cols)
	}
	if cols[0].Href != "/calendars/user/personal/" {
		t.Fatalf("unexpected href: %q", cols[0].Href)
	}
	if cols[0].DisplayName != "Personal" {
		t.Fatalf("unexpected displayName: %q", cols[0].DisplayName)
	}
}

func TestResolveAbsoluteURL(t *testing.T) {
	c, err := New("https://dav.example.com", "u", "p")
	if err != nil {
		t.Fatal(err)
	}
	abs := "https://other.example.com/path"
	got := c.resolve(abs)
	if got != abs {
		t.Fatalf("expected %q, got %q", abs, got)
	}
}

func TestPutIfNoneMatch(t *testing.T) {
	var gotHeader string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("If-None-Match")
		w.WriteHeader(201)
	})
	defer srv.Close()

	err := c.Put(context.Background(), "/cal/new.ics", "text/calendar", "", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	if gotHeader != "*" {
		t.Fatalf("expected If-None-Match: *, got %q", gotHeader)
	}
}

func TestMapHTTPError(t *testing.T) {
	cases := []struct {
		code int
		want error
	}{
		{404, ErrNotFound},
		{409, ErrConflict},
		{412, ErrPreconditionFailed},
	}
	for _, tc := range cases {
		if got := mapHTTPError(tc.code); got != tc.want {
			t.Errorf("mapHTTPError(%d) = %v, want %v", tc.code, got, tc.want)
		}
	}
}
