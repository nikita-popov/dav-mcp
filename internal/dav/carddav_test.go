package dav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const abQueryResp = `<?xml version="1.0"?>
<multistatus xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:carddav">
  <response>
    <href>/addressbooks/user/contacts/alice.vcf</href>
    <propstat>
      <prop>
        <getetag>"e1"</getetag>
        <c:address-data>BEGIN:VCARD
VERSION:3.0
FN:Alice Smith
EMAIL:alice@example.com
END:VCARD
</c:address-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
  <response>
    <href>/addressbooks/user/contacts/bob.vcf</href>
    <propstat>
      <prop>
        <getetag>"e2"</getetag>
        <c:address-data>BEGIN:VCARD
VERSION:3.0
FN:Bob Jones
EMAIL:bob@example.com
END:VCARD
</c:address-data>
      </prop>
      <status>HTTP/1.1 200 OK</status>
    </propstat>
  </response>
</multistatus>`

func TestQueryContacts_ReturnsTwoVCards(t *testing.T) {
	var gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(abQueryResp))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	vCards, err := QueryContacts(context.Background(), c, "/addressbooks/user/contacts/")
	if err != nil {
		t.Fatalf("QueryContacts: %v", err)
	}
	if gotMethod != "REPORT" {
		t.Errorf("method=%q, want REPORT", gotMethod)
	}
	if len(vCards) != 2 {
		t.Errorf("expected 2 vCards, got %d", len(vCards))
	}
	if !strings.Contains(vCards[0], "Alice Smith") {
		t.Errorf("first vCard missing Alice")
	}
}

func TestQueryContacts_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	vCards, err := QueryContacts(context.Background(), c, "/ab/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vCards) != 0 {
		t.Errorf("expected 0, got %d", len(vCards))
	}
}

func TestPutContact_SendsPUT(t *testing.T) {
	var gotMethod, gotPath, gotCT string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(201)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	err := PutContact(context.Background(), c, "/ab/", "uid123", "BEGIN:VCARD\r\nEND:VCARD\r\n", "")
	if err != nil {
		t.Fatalf("PutContact: %v", err)
	}
	if gotMethod != "PUT" {
		t.Errorf("method=%q", gotMethod)
	}
	if gotPath != "/ab/uid123.vcf" {
		t.Errorf("path=%q", gotPath)
	}
	if !strings.HasPrefix(gotCT, "text/vcard") {
		t.Errorf("Content-Type=%q", gotCT)
	}
}

func TestDeleteContact_SendsDELETE(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	err := DeleteContact(context.Background(), c, "/ab/", "uid123")
	if err != nil {
		t.Fatalf("DeleteContact: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method=%q", gotMethod)
	}
	if gotPath != "/ab/uid123.vcf" {
		t.Errorf("path=%q", gotPath)
	}
}

func TestQueryContacts_ChecksREPORTBody(t *testing.T) {
	var gotBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(207)
		w.Write([]byte(`<?xml version="1.0"?><multistatus xmlns="DAV:"></multistatus>`))
	}))
	defer srv.Close()

	c, _ := New(srv.URL, "u", "p")
	QueryContacts(context.Background(), c, "/ab/") //nolint:errcheck
	if !strings.Contains(gotBody, "addressbook-query") {
		t.Errorf("REPORT body missing addressbook-query, got: %s", gotBody)
	}
	if !strings.Contains(gotBody, "address-data") {
		t.Errorf("REPORT body missing address-data")
	}
}
