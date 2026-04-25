package tools_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

const testVCard = `BEGIN:VCARD\r\nVERSION:3.0\r\nUID:uid-alice-001\r\nFN:Alice Smith\r\nEMAIL:alice@example.com\r\nEND:VCARD\r\n`

const cardDAVReport = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
  <response>
    <href>/addressbooks/user/default/alice.vcf</href>
    <propstat>
      <prop>
        <getetag>"etag-alice"</getetag>
        <card:address-data>BEGIN:VCARD
VERSION:3.0
UID:uid-alice-001
FN:Alice Smith
EMAIL:alice@example.com
END:VCARD
</card:address-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

const abCollections = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response>
    <href>/addressbooks/user/default/</href>
    <propstat>
      <prop><displayname>Default</displayname><resourcetype><collection/></resourcetype></prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

// cardDAVServer returns a test CardDAV server.
// abHandler is called for any request whose path has prefix /addressbooks/user/default.
func cardDAVServer(t *testing.T, abHandler http.HandlerFunc) *httptest.Server {
	t.Helper()

	const principalResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:">
  <response><href>/</href>
    <propstat><prop><current-user-principal><href>/principals/user/</href></current-user-principal></prop>
    <status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

	const homesResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav" xmlns:card="urn:ietf:params:xml:ns:carddav">
  <response><href>/principals/user/</href>
    <propstat><prop>
      <c:calendar-home-set><href>/calendars/user/</href></c:calendar-home-set>
      <card:addressbook-home-set><href>/addressbooks/user/</href></card:addressbook-home-set>
    </prop><status>HTTP/1.1 200 OK</status></propstat>
  </response>
</multistatus>`

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")

		if abHandler != nil && strings.HasPrefix(r.URL.Path, "/addressbooks/user/default") {
			abHandler(w, r)
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/addressbooks"):
			w.WriteHeader(207)
			w.Write([]byte(abCollections))
		case strings.HasPrefix(r.URL.Path, "/calendars"):
			w.WriteHeader(207)
			w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
		case strings.HasPrefix(r.URL.Path, "/principals"):
			w.WriteHeader(207)
			w.Write([]byte(homesResp))
		default:
			w.WriteHeader(207)
			w.Write([]byte(principalResp))
		}
	}))
}

func connectCardDAV(t *testing.T, abHandler http.HandlerFunc) (config.Config, func()) {
	t.Helper()
	srv := cardDAVServer(t, abHandler)
	cfg := config.Config{
		Accounts: []config.Account{{
			Name:     "default",
			URL:      srv.URL,
			Username: "user",
			Password: "pass",
		}},
	}
	_, err := dav.Connect(context.Background(), "default", srv.URL, "user", "pass")
	if err != nil {
		srv.Close()
		t.Fatalf("dav.Connect: %v", err)
	}
	return cfg, func() { srv.Close() }
}

func contactsServer(t *testing.T, cfg config.Config) *mcp.Server {
	t.Helper()
	s := mcp.NewServer("test", "0")
	tools.RegisterContacts(s, cfg)
	return s
}

// ---- contacts_list ----------------------------------------------------------

func TestContactsList(t *testing.T) {
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "REPORT" {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(207)
			w.Write([]byte(cardDAVReport))
			return
		}
		w.WriteHeader(405)
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	res, err := s.CallTool(context.Background(), "contacts_list", map[string]any{
		"addressbook": "/addressbooks/user/default/",
	})
	if err != nil {
		t.Fatalf("contacts_list: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !strings.Contains(toolText(t, res), "Alice Smith") {
		t.Errorf("expected Alice Smith in output, got: %s", toolText(t, res))
	}
}

// ---- contacts_get -----------------------------------------------------------

func TestContactsGet_Found(t *testing.T) {
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(cardDAVReport))
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	res, err := s.CallTool(context.Background(), "contacts_get", map[string]any{
		"uid":         "uid-alice-001",
		"addressbook": "/addressbooks/user/default/",
	})
	if err != nil {
		t.Fatalf("contacts_get: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !strings.Contains(toolText(t, res), "alice@example.com") {
		t.Errorf("expected email in output, got: %s", toolText(t, res))
	}
}

func TestContactsGet_NotFound(t *testing.T) {
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(cardDAVReport))
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	_, err := s.CallTool(context.Background(), "contacts_get", map[string]any{
		"uid":         "no-such-uid",
		"addressbook": "/addressbooks/user/default/",
	})
	if err == nil {
		t.Fatal("expected error for unknown UID")
	}
}

// ---- contacts_search --------------------------------------------------------

func TestContactsSearch(t *testing.T) {
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(cardDAVReport))
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	res, err := s.CallTool(context.Background(), "contacts_search", map[string]any{
		"query":       "alice",
		"addressbook": "/addressbooks/user/default/",
	})
	if err != nil {
		t.Fatalf("contacts_search: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !strings.Contains(toolText(t, res), "1 result") {
		t.Errorf("expected 1 result, got: %s", toolText(t, res))
	}
}

// ---- contacts_create --------------------------------------------------------

func TestContactsCreate(t *testing.T) {
	var putCalled bool
	// Handler receives all /addressbooks/user/default/* — including /<uid>.vcf PUT.
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			putCalled = true
			w.WriteHeader(201)
			return
		}
		w.WriteHeader(405)
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	res, err := s.CallTool(context.Background(), "contacts_create", map[string]any{
		"name":        "Bob Jones",
		"email":       "bob@example.com",
		"addressbook": "/addressbooks/user/default/",
	})
	if err != nil {
		t.Fatalf("contacts_create: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if !putCalled {
		t.Error("expected PUT on server")
	}
	if !strings.Contains(toolText(t, res), "Bob Jones") {
		t.Errorf("unexpected output: %s", toolText(t, res))
	}
}

// ---- contacts_delete --------------------------------------------------------

func TestContactsDelete(t *testing.T) {
	var deletedPath string
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "REPORT":
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(207)
			w.Write([]byte(cardDAVReport))
		case "DELETE":
			deletedPath = r.URL.Path
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	res, err := s.CallTool(context.Background(), "contacts_delete", map[string]any{
		"uid":         "uid-alice-001",
		"addressbook": "/addressbooks/user/default/",
	})
	if err != nil {
		t.Fatalf("contacts_delete: %v", err)
	}
	if toolIsError(res) {
		t.Fatalf("tool error: %s", toolText(t, res))
	}
	if deletedPath == "" {
		t.Error("expected DELETE to be called on server")
	}
	if !strings.Contains(toolText(t, res), "uid-alice-001") {
		t.Errorf("expected UID in output, got: %s", toolText(t, res))
	}
}

func TestContactsDelete_NotFound(t *testing.T) {
	cfg, cleanup := connectCardDAV(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(cardDAVReport))
	}))
	defer cleanup()

	s := contactsServer(t, cfg)
	_, err := s.CallTool(context.Background(), "contacts_delete", map[string]any{
		"uid":         "ghost-uid",
		"addressbook": "/addressbooks/user/default/",
	})
	if err == nil {
		t.Fatal("expected error for unknown UID")
	}
}
