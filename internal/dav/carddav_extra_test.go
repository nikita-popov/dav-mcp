package dav

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const contactsFullResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:carddav">
  <response>
    <href>/addressbooks/user/default/c1.vcf</href>
    <propstat>
      <prop>
        <getetag>"etag-c1"</getetag>
        <c:address-data>BEGIN:VCARD
VERSION:3.0
UID:c1@test
FN:Alice Smith
EMAIL:alice@example.com
END:VCARD
</c:address-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestQueryContactsFull_ReturnsOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "REPORT" {
			t.Errorf("method=%q, want REPORT", r.Method)
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(contactsFullResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	contacts, err := QueryContactsFull(context.Background(), c, "/addressbooks/user/default/")
	if err != nil {
		t.Fatalf("QueryContactsFull: %v", err)
	}
	if len(contacts) != 1 {
		t.Errorf("expected 1, got %d", len(contacts))
	}
	if contacts[0].Contact.FN != "Alice Smith" {
		t.Errorf("FN=%q", contacts[0].Contact.FN)
	}
}

func TestQueryContactsFull_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	contacts, err := QueryContactsFull(context.Background(), c, "/addressbooks/user/default/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(contacts) != 0 {
		t.Errorf("expected 0, got %d", len(contacts))
	}
}

func TestPutContactHref_SendsPUT(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/addressbooks") {
			gotPath = r.URL.Path
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	vcfData := "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:c1@test\r\nFN:Alice Smith\r\nEND:VCARD\r\n"
	err := PutContactHref(context.Background(), c, "/addressbooks/user/default/c1.vcf", vcfData, "\"etag-c1\"")
	if err != nil {
		t.Fatalf("PutContactHref: %v", err)
	}
	if gotPath != "/addressbooks/user/default/c1.vcf" {
		t.Errorf("path=%q", gotPath)
	}
}
