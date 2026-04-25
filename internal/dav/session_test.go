package dav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

// multiHandler routes PROPFIND requests by path prefix, longest prefix first.
// Using a plain map caused non-deterministic routing because Go map iteration
// order is random — "/" could match before "/calendars".
func multiHandler(routes map[string]string) http.HandlerFunc {
	// sort prefixes longest → shortest so more-specific routes win
	prefixes := make([]string, 0, len(routes))
	for p := range routes {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return len(prefixes[i]) > len(prefixes[j])
	})

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		for _, prefix := range prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				w.WriteHeader(207)
				w.Write([]byte(routes[prefix]))
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

// clearStore resets the global session store between tests.
func clearStore() {
	mu.Lock()
	defer mu.Unlock()
	store = map[string]*Session{}
}

func TestConnect_Success(t *testing.T) {
	clearStore()
	srv := fullDiscoveryServer(t)
	defer srv.Close()

	session, err := Connect(context.Background(), "default", srv.URL, "user", "pass")
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
	clearStore()
	srv := fullDiscoveryServer(t)
	defer srv.Close()

	Connect(context.Background(), "default", srv.URL, "user", "pass") //nolint:errcheck
	if Get("") == nil {
		t.Fatal("singleton not set after Connect")
	}
}

func TestConnect_BadURL(t *testing.T) {
	_, err := Connect(context.Background(), "default", "://bad", "u", "p")
	if err == nil {
		t.Fatal("expected error for bad URL")
	}
}

func TestConnect_PrincipalNotFound(t *testing.T) {
	clearStore()
	emptySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer emptySrv.Close()

	_, err := Connect(context.Background(), "default", emptySrv.URL, "u", "p")
	if err == nil {
		t.Fatal("expected error when principal not found")
	}
}

func TestGet_NilBeforeConnect(t *testing.T) {
	clearStore()
	if Get("") != nil {
		t.Fatal("expected nil before Connect")
	}
}

func TestConnect_StoresAddressbookHome(t *testing.T) {
	clearStore()

	principalWithAB := `<?xml version="1.0"?>
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		switch {
		case strings.HasPrefix(r.URL.Path, "/calendars"):
			w.Write([]byte(collectionsResp))
		case strings.HasPrefix(r.URL.Path, "/principals"):
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "addressbook-home-set") {
				w.Write([]byte(abHomeResp))
			} else {
				w.Write([]byte(calHomeResp))
			}
		default:
			w.Write([]byte(principalWithAB))
		}
	}))
	defer srv.Close()

	session, err := Connect(context.Background(), "default", srv.URL, "u", "p")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if session.AddressbookHome != "/addressbooks/user/" {
		t.Errorf("AddressbookHome=%q", session.AddressbookHome)
	}
}
