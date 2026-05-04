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

// tzidEvent mirrors what Yandex Calendar actually sends: DTSTART with a
// TZID parameter and a local (non-UTC) datetime value.
const tzidEvent = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:tz@x\r\nSUMMARY:TZ Meeting\r\nDTSTART;TZID=Asia/Yekaterinburg:20260504T143000\r\nDTEND;TZID=Asia/Yekaterinburg:20260504T150000\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

// recurringTZIDEvent combines RRULE with TZID — the combination that was
// silently broken: StartTZ was empty and Start was parsed as floating UTC.
const recurringTZIDEvent = "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nUID:rectز@x\r\nSUMMARY:Daily standup\r\nDTSTART;TZID=Europe/Moscow:20260501T100000\r\nDTEND;TZID=Europe/Moscow:20260501T101500\r\nRRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

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

// TestParseEvents_TZID verifies that DTSTART;TZID=Asia/Yekaterinburg is
// parsed as the correct UTC instant and that StartTZ/EndTZ are populated.
func TestParseEvents_TZID(t *testing.T) {
	evs := ParseEvents(tzidEvent)
	if len(evs) != 1 {
		t.Fatalf("expected 1, got %d", len(evs))
	}
	e := evs[0]

	if e.StartTZ != "Asia/Yekaterinburg" {
		t.Errorf("StartTZ=%q, want Asia/Yekaterinburg", e.StartTZ)
	}
	if e.EndTZ != "Asia/Yekaterinburg" {
		t.Errorf("EndTZ=%q, want Asia/Yekaterinburg", e.EndTZ)
	}

	// Asia/Yekaterinburg is UTC+5; 14:30 local = 09:30 UTC.
	wantStart := time.Date(2026, 5, 4, 9, 30, 0, 0, time.UTC)
	if !e.Start.Equal(wantStart) {
		t.Errorf("Start=%v, want %v (UTC)", e.Start, wantStart)
	}
	wantEnd := time.Date(2026, 5, 4, 10, 0, 0, 0, time.UTC)
	if !e.End.Equal(wantEnd) {
		t.Errorf("End=%v, want %v (UTC)", e.End, wantEnd)
	}
}

// TestParseEvents_RecurringWithTZID verifies that an event combining RRULE
// and TZID has both RRule populated and Start parsed in the correct timezone.
func TestParseEvents_RecurringWithTZID(t *testing.T) {
	evs := ParseEvents(recurringTZIDEvent)
	if len(evs) != 1 {
		t.Fatalf("expected 1, got %d", len(evs))
	}
	e := evs[0]

	if e.RRule != "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR" {
		t.Errorf("RRule=%q", e.RRule)
	}
	if e.StartTZ != "Europe/Moscow" {
		t.Errorf("StartTZ=%q, want Europe/Moscow", e.StartTZ)
	}

	// Europe/Moscow is UTC+3; 10:00 local = 07:00 UTC.
	wantStart := time.Date(2026, 5, 1, 7, 0, 0, 0, time.UTC)
	if !e.Start.Equal(wantStart) {
		t.Errorf("Start=%v, want %v (UTC)", e.Start, wantStart)
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
