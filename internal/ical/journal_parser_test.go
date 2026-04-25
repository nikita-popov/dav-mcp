package ical

import (
	"testing"
	"time"
)

const singleJournal = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VJOURNAL\r\nUID:jrn-1@dav-mcp\r\nSUMMARY:Sprint retrospective\r\nDESCRIPTION:Went well overall\r\nDTSTART;VALUE=DATE:20260425\r\nSTATUS:FINAL\r\nEND:VJOURNAL\r\nEND:VCALENDAR\r\n"

const twoJournals = "BEGIN:VCALENDAR\r\nBEGIN:VJOURNAL\r\nUID:jrn1@x\r\nSUMMARY:Day one\r\nDTSTART;VALUE=DATE:20260401\r\nEND:VJOURNAL\r\nBEGIN:VJOURNAL\r\nUID:jrn2@x\r\nSUMMARY:Day two\r\nDTSTART;VALUE=DATE:20260402\r\nEND:VJOURNAL\r\nEND:VCALENDAR\r\n"

const journalWithTimestamp = "BEGIN:VCALENDAR\r\nBEGIN:VJOURNAL\r\nUID:jrn3@x\r\nSUMMARY:Meeting notes\r\nDTSTART:20260425T140000Z\r\nEND:VJOURNAL\r\nEND:VCALENDAR\r\n"

const mixedWithJournal = "BEGIN:VCALENDAR\r\nBEGIN:VTODO\r\nUID:td@x\r\nSUMMARY:Task\r\nEND:VTODO\r\nBEGIN:VJOURNAL\r\nUID:jrn@x\r\nSUMMARY:Journal\r\nDTSTART;VALUE=DATE:20260425\r\nEND:VJOURNAL\r\nEND:VCALENDAR\r\n"

func TestParseJournals_Single(t *testing.T) {
	journals := ParseJournals(singleJournal)
	if len(journals) != 1 {
		t.Fatalf("expected 1, got %d", len(journals))
	}
	j := journals[0]
	if j.UID != "jrn-1@dav-mcp" {
		t.Errorf("UID=%q", j.UID)
	}
	if j.Summary != "Sprint retrospective" {
		t.Errorf("Summary=%q", j.Summary)
	}
	if j.Description != "Went well overall" {
		t.Errorf("Description=%q", j.Description)
	}
	if j.Status != "FINAL" {
		t.Errorf("Status=%q", j.Status)
	}
	wantDate := time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC)
	if !j.Date.Equal(wantDate) {
		t.Errorf("Date=%v, want %v", j.Date, wantDate)
	}
}

func TestParseJournals_Two(t *testing.T) {
	journals := ParseJournals(twoJournals)
	if len(journals) != 2 {
		t.Fatalf("expected 2, got %d", len(journals))
	}
	if journals[0].UID != "jrn1@x" || journals[1].UID != "jrn2@x" {
		t.Errorf("UIDs: %q %q", journals[0].UID, journals[1].UID)
	}
}

func TestParseJournals_WithTimestamp(t *testing.T) {
	journals := ParseJournals(journalWithTimestamp)
	if len(journals) != 1 {
		t.Fatalf("expected 1, got %d", len(journals))
	}
	d := journals[0].Date
	if d.Year() != 2026 || d.Month() != 4 || d.Day() != 25 {
		t.Errorf("Date=%v", d)
	}
}

func TestParseJournals_IgnoresVTODO(t *testing.T) {
	journals := ParseJournals(mixedWithJournal)
	if len(journals) != 1 {
		t.Fatalf("expected 1 journal, got %d", len(journals))
	}
	if journals[0].UID != "jrn@x" {
		t.Errorf("UID=%q", journals[0].UID)
	}
}

func TestParseJournals_Empty(t *testing.T) {
	if journals := ParseJournals(""); len(journals) != 0 {
		t.Errorf("expected 0, got %d", len(journals))
	}
}

// Round-trip: BuildJournal → ParseJournals
func TestBuildJournal_RoundTrip(t *testing.T) {
	orig := Journal{
		UID:         "rt-jrn@dav-mcp",
		Summary:     "Daily standup notes",
		Description: "Discussed blockers\nand plans",
		Date:        time.Date(2026, 4, 25, 0, 0, 0, 0, time.UTC),
	}
	data := BuildJournal(orig)
	journals := ParseJournals(data)
	if len(journals) != 1 {
		t.Fatalf("expected 1, got %d", len(journals))
	}
	got := journals[0]
	if got.UID != orig.UID {
		t.Errorf("UID: got %q, want %q", got.UID, orig.UID)
	}
	if got.Summary != orig.Summary {
		t.Errorf("Summary: got %q, want %q", got.Summary, orig.Summary)
	}
	if got.Description != orig.Description {
		t.Errorf("Description: got %q, want %q", got.Description, orig.Description)
	}
	if !got.Date.Equal(orig.Date) {
		t.Errorf("Date: got %v, want %v", got.Date, orig.Date)
	}
}
