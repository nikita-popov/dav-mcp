package ical

import (
	"testing"
	"time"
)

const singleEvent = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:abc-123@dav-mcp\r\nSUMMARY:Team meeting\r\nDTSTART:20260501T100000Z\r\nDTEND:20260501T110000Z\r\nDESCRIPTION:Weekly sync\r\nLOCATION:Room 42\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

const twoEvents = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:ev1@x\r\nSUMMARY:First\r\nDTSTART:20260501T090000Z\r\nDTEND:20260501T100000Z\r\nEND:VEVENT\r\nBEGIN:VEVENT\r\nUID:ev2@x\r\nSUMMARY:Second\r\nDTSTART:20260502T090000Z\r\nDTEND:20260502T100000Z\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

const allDayEvent = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:ad@x\r\nSUMMARY:Holiday\r\nDTSTART;VALUE=DATE:20260601\r\nDTEND;VALUE=DATE:20260602\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

const foldedEvent = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:fold@x\r\nSUMMARY:A very long summa\r\n ry that is folded\r\nDTSTART:20260501T100000Z\r\nDTEND:20260501T110000Z\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

const recurringEvent = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:rec@x\r\nSUMMARY:Standup\r\nDTSTART:20260501T090000Z\r\nDTEND:20260501T091500Z\r\nRRULE:FREQ=DAILY;BYDAY=MO,TU,WE,TH,FR\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

func TestParseEvents_Single(t *testing.T) {
	evs := ParseEvents(singleEvent)
	if len(evs) != 1 {
		t.Fatalf("expected 1, got %d", len(evs))
	}
	e := evs[0]
	if e.UID != "abc-123@dav-mcp" {
		t.Errorf("UID=%q", e.UID)
	}
	if e.Summary != "Team meeting" {
		t.Errorf("Summary=%q", e.Summary)
	}
	if e.Description != "Weekly sync" {
		t.Errorf("Description=%q", e.Description)
	}
	if e.Location != "Room 42" {
		t.Errorf("Location=%q", e.Location)
	}
	wantStart := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	if !e.Start.Equal(wantStart) {
		t.Errorf("Start=%v", e.Start)
	}
}

func TestParseEvents_Two(t *testing.T) {
	evs := ParseEvents(twoEvents)
	if len(evs) != 2 {
		t.Fatalf("expected 2, got %d", len(evs))
	}
	if evs[0].UID != "ev1@x" || evs[1].UID != "ev2@x" {
		t.Errorf("UIDs: %q %q", evs[0].UID, evs[1].UID)
	}
}

func TestParseEvents_AllDay(t *testing.T) {
	evs := ParseEvents(allDayEvent)
	if len(evs) != 1 {
		t.Fatalf("expected 1, got %d", len(evs))
	}
	if evs[0].Start.Year() != 2026 || evs[0].Start.Month() != 6 || evs[0].Start.Day() != 1 {
		t.Errorf("Start=%v", evs[0].Start)
	}
}

func TestParseEvents_Folded(t *testing.T) {
	evs := ParseEvents(foldedEvent)
	if len(evs) != 1 {
		t.Fatalf("expected 1, got %d", len(evs))
	}
	if evs[0].Summary != "A very long summary that is folded" {
		t.Errorf("Summary=%q", evs[0].Summary)
	}
}

func TestParseEvents_Recurring(t *testing.T) {
	evs := ParseEvents(recurringEvent)
	if len(evs) != 1 {
		t.Fatalf("expected 1, got %d", len(evs))
	}
	if evs[0].RRule != "FREQ=DAILY;BYDAY=MO,TU,WE,TH,FR" {
		t.Errorf("RRule=%q", evs[0].RRule)
	}
}

func TestParseEvents_Empty(t *testing.T) {
	if evs := ParseEvents(""); len(evs) != 0 {
		t.Errorf("expected 0, got %d", len(evs))
	}
}

func TestUnescape(t *testing.T) {
	cases := []struct{ in, want string }{
		{`hello\nworld`, "hello\nworld"},
		{`a\,b`, "a,b"},
		{`a\;b`, "a;b"},
		{`a\\b`, `a\b`},
	}
	for _, c := range cases {
		if got := unescape(c.in); got != c.want {
			t.Errorf("unescape(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
