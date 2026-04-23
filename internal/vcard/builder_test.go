package vcard

import (
	"strings"
	"testing"
)

func has(t *testing.T, label, src, sub string) {
	t.Helper()
	if !strings.Contains(src, sub) {
		t.Errorf("%s: expected %q\ngot:\n%s", label, sub, src)
	}
}

func hasNot(t *testing.T, label, src, sub string) {
	t.Helper()
	if strings.Contains(src, sub) {
		t.Errorf("%s: must NOT contain %q\ngot:\n%s", label, sub, src)
	}
}

func TestBuild_Basic(t *testing.T) {
	c := Build(Contact{
		UID:   "test-uid@dav-mcp",
		FN:    "John Doe",
		Email: "john@example.com",
		Phone: "+79991234567",
		Org:   "ACME",
		Notes: "test note",
	})
	has(t, "basic", c, "BEGIN:VCARD")
	has(t, "basic", c, "VERSION:4.0")
	has(t, "basic", c, "UID:test-uid@dav-mcp")
	has(t, "basic", c, "FN:John Doe")
	has(t, "basic", c, "EMAIL:john@example.com")
	has(t, "basic", c, "TEL:+79991234567")
	has(t, "basic", c, "ORG:ACME")
	has(t, "basic", c, "NOTE:test note")
	has(t, "basic", c, "END:VCARD")
}

func TestBuild_OptionalFieldsOmitted(t *testing.T) {
	c := Build(Contact{UID: "min@dav-mcp", FN: "Alice"})
	hasNot(t, "optional", c, "EMAIL:")
	hasNot(t, "optional", c, "TEL:")
	hasNot(t, "optional", c, "ORG:")
	hasNot(t, "optional", c, "NOTE:")
}

func TestBuild_CRLF(t *testing.T) {
	c := Build(Contact{UID: "crlf@dav-mcp", FN: "X"})
	if !strings.Contains(c, "\r\n") {
		t.Error("output must use CRLF line endings")
	}
}

func TestBuild_Folding(t *testing.T) {
	long := strings.Repeat("B", 200)
	c := Build(Contact{UID: "fold@dav-mcp", FN: long})
	for _, l := range strings.Split(c, "\r\n") {
		if len(l) > 75 {
			t.Errorf("line exceeds 75 chars (%d): %q", len(l), l)
		}
	}
}

func TestBuild_Escape(t *testing.T) {
	c := Build(Contact{
		UID:   "esc@dav-mcp",
		FN:    "Doe\\, John\\; Jr",
		Notes: "line1\nline2",
	})
	has(t, "escape-note", c, `NOTE:line1\nline2`)
}

func TestParseFN(t *testing.T) {
	raw := Build(Contact{UID: "p@dav-mcp", FN: "Parse Test"})
	got := ParseFN(raw)
	if got != "Parse Test" {
		t.Errorf("ParseFN: expected %q, got %q", "Parse Test", got)
	}
}

func TestParseUID(t *testing.T) {
	raw := Build(Contact{UID: "uid-parse@dav-mcp", FN: "X"})
	got := ParseUID(raw)
	if got != "uid-parse@dav-mcp" {
		t.Errorf("ParseUID: expected %q, got %q", "uid-parse@dav-mcp", got)
	}
}

func TestParseFN_Folded(t *testing.T) {
	raw := "BEGIN:VCARD\r\nVERSION:4.0\r\nFN:Very Long Na\r\n me Here\r\nEND:VCARD\r\n"
	got := ParseFN(raw)
	if got != "Very Long Name Here" {
		t.Errorf("ParseFN folded: expected %q, got %q", "Very Long Name Here", got)
	}
}

func TestNewUID_Unique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		uid := NewUID()
		if seen[uid] {
			t.Fatalf("duplicate UID: %s", uid)
		}
		seen[uid] = true
		if !strings.HasSuffix(uid, "@dav-mcp") {
			t.Errorf("unexpected UID format: %s", uid)
		}
	}
}
