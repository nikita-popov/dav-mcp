package vcard

import (
	"testing"
)

const singleVCard = "BEGIN:VCARD\r\nVERSION:3.0\r\nUID:alice-uid\r\nFN:Alice Smith\r\nEMAIL:alice@example.com\r\nTEL:+1-555-0100\r\nORG:Acme Corp\r\nNOTE:Test note\r\nEND:VCARD\r\n"

const twoVCards = "BEGIN:VCARD\r\nFN:Alice\r\nUID:uid1\r\nEND:VCARD\r\nBEGIN:VCARD\r\nFN:Bob\r\nUID:uid2\r\nEND:VCARD\r\n"

const multiEmail = "BEGIN:VCARD\r\nFN:Carol\r\nEMAIL;TYPE=work:carol@work.com\r\nEMAIL;TYPE=home:carol@home.com\r\nEND:VCARD\r\n"

const foldedVCard = "BEGIN:VCARD\r\nFN:A very long na\r\n me that is folded\r\nEND:VCARD\r\n"

func TestParseContacts_Single(t *testing.T) {
	cs := ParseContacts(singleVCard)
	if len(cs) != 1 {
		t.Fatalf("expected 1, got %d", len(cs))
	}
	c := cs[0]
	if c.UID != "alice-uid" {
		t.Errorf("UID=%q", c.UID)
	}
	if c.FN != "Alice Smith" {
		t.Errorf("FN=%q", c.FN)
	}
	if len(c.Email) != 1 || c.Email[0] != "alice@example.com" {
		t.Errorf("Email=%v", c.Email)
	}
	if len(c.Phone) != 1 || c.Phone[0] != "+1-555-0100" {
		t.Errorf("Phone=%v", c.Phone)
	}
	if c.Org != "Acme Corp" {
		t.Errorf("Org=%q", c.Org)
	}
	if c.Note != "Test note" {
		t.Errorf("Note=%q", c.Note)
	}
}

func TestParseContacts_Two(t *testing.T) {
	cs := ParseContacts(twoVCards)
	if len(cs) != 2 {
		t.Fatalf("expected 2, got %d", len(cs))
	}
	if cs[0].UID != "uid1" || cs[1].UID != "uid2" {
		t.Errorf("UIDs: %q %q", cs[0].UID, cs[1].UID)
	}
}

func TestParseContacts_MultiEmail(t *testing.T) {
	cs := ParseContacts(multiEmail)
	if len(cs) != 1 {
		t.Fatalf("expected 1, got %d", len(cs))
	}
	if len(cs[0].Email) != 2 {
		t.Errorf("expected 2 emails, got %v", cs[0].Email)
	}
}

func TestParseContacts_Folded(t *testing.T) {
	cs := ParseContacts(foldedVCard)
	if len(cs) != 1 {
		t.Fatalf("expected 1, got %d", len(cs))
	}
	if cs[0].FN != "A very long name that is folded" {
		t.Errorf("FN=%q", cs[0].FN)
	}
}

func TestParseContacts_Empty(t *testing.T) {
	if cs := ParseContacts(""); len(cs) != 0 {
		t.Errorf("expected 0, got %d", len(cs))
	}
}
