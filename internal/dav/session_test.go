package dav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// multiHandler routes PROPFIND requests by path prefix.
func multiHandler(routes map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		for prefix, body := range routes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				w.WriteHeader(207)
				w.Write([]byte(body))
				return
			}
		}
		http.NotFound(w, r)
	}
}

// fullDiscoveryServer wires up all three discovery steps:
// /            → principal
// /principals/ → calHome
// /calendars/  → collections
func fullDiscoveryServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(multiHandler(map[string]string{
		"/principals": calHomeResp,
		"/":           principalResp,
		"/calendars":  collectionsResp,
	}))
}

func TestConnect_Success(t *testing.T) {
	setSession(nil)
	srv := fullDiscoveryServer(t)
	defer srv.Close()

	session, err := Connect(context.Background(), srv.URL, "user", "pass")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if session.CalendarHome != "/calendars/user/" {
		t.Errorf("CalendarHome=%q", session.CalendarHome)
	}
	if len(session.Calendars) != 1 {
		t.Errorf("expected 1 calendar, got %d", len(session.Calendars))
	}
}

func TestConnect_SetsSingleton(t *testing.T) {
	setSession(nil)
	srv := fullDiscoveryServer(t)
	defer srv.Close()

	Connect(context.Background(), srv.URL, "user", "pass") //nolint:errcheck
	if Get() == nil {
		t.Fatal("singleton not set after Connect")
	}
}

func TestConnect_BadURL(t *testing.T) {
	_, err := Connect(context.Background(), "://bad", "u", "p")
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

func TestConnect_PrincipalNotFound(t *testing.T) {
	setSession(nil)
	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer emptySrv.Close()

	_, err := Connect(context.Background(), emptySrv.URL, "u", "p")
	if err == nil {
		t.Fatal("expected error when principal not found")
	}
}

func TestGet_NilBeforeConnect(t *testing.T) {
	setSession(nil)
	if Get() != nil {
		t.Fatal("expected nil before Connect")
	}
}

func TestConnect_StoresAddressbookHome(t *testing.T) {
	setSession(nil)
	// principal response also includes addressbook-home-set
	principalWithAB := `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
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

	abHomeResp := `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
  <response>
    <href>/principals/user/</href>
    <propstat>
      <prop>
        <card:addressbook-home-set><href>/addressbooks/user/</href></card:addressbook-home-set>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

	var reqCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		reqCount++
		switch {
		case strings.HasPrefix(r.URL.Path, "/principals"):
			// second call to /principals: return calHome or abHome based on request body
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "addressbook-home-set") {
				w.Write([]byte(abHomeResp))
			} else {
				w.Write([]byte(calHomeResp))
			}
		case strings.HasPrefix(r.URL.Path, "/calendars"):
			w.Write([]byte(collectionsResp))
		default:
			w.Write([]byte(principalWithAB))
		}
	}))
	defer srv.Close()

	session, err := Connect(context.Background(), srv.URL, "u", "p")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if session.AddressbookHome != "/addressbooks/user/" {
		t.Errorf("AddressbookHome=%q", session.AddressbookHome)
	}
}
