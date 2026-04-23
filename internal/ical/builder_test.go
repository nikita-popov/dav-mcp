package ical

import (
	"strings"
	"testing"
	"time"
)

func mustContain(t *testing.T, label, src, sub string) {
	t.Helper()
	if !strings.Contains(src, sub) {
		t.Errorf("%s: expected to contain %q\ngot:\n%s", label, sub, src)
	}
}

func mustNotContain(t *testing.T, label, src, sub string) {
	t.Helper()
	if strings.Contains(src, sub) {
		t.Errorf("%s: must NOT contain %q\ngot:\n%s", label, sub, src)
	}
}

var (
	start = time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	end   = time.Date(2026, 4, 25, 11, 0, 0, 0, time.UTC)
	due   = time.Date(2026, 4, 30, 18, 0, 0, 0, time.UTC)
	day   = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
)

func TestBuildEvent_Basic(t *testing.T) {
	out := BuildEvent(Event{
		UID:     "test-uid@dav-mcp",
		Summary: "Team meeting",
		Start:   start,
		End:     end,
	})
	mustContain(t, "event", out, "BEGIN:VCALENDAR")
	mustContain(t, "event", out, "BEGIN:VEVENT")
	mustContain(t, "event", out, "UID:test-uid@dav-mcp")
	mustContain(t, "event", out, "DTSTART:20260425T100000Z")
	mustContain(t, "event", out, "DTEND:20260425T110000Z")
	mustContain(t, "event", out, "SUMMARY:Team meeting")
	mustContain(t, "event", out, "END:VEVENT")
	mustContain(t, "event", out, "END:VCALENDAR")
	mustContain(t, "event", out, "PRODID:-//dav-mcp//EN")
	mustNotContain(t, "event", out, "DESCRIPTION:")
	mustNotContain(t, "event", out, "RRULE:")
}

func TestBuildEvent_AllDay(t *testing.T) {
	out := BuildEvent(Event{
		UID:     "allday@dav-mcp",
		Summary: "Holiday",
		Start:   day,
		End:     day.AddDate(0, 0, 1),
		AllDay:  true,
	})
	mustContain(t, "allday", out, "DTSTART;VALUE=DATE:20260501")
	mustContain(t, "allday", out, "DTEND;VALUE=DATE:20260502")
}

func TestBuildEvent_Recurring(t *testing.T) {
	out := BuildEvent(Event{
		UID:     "recur@dav-mcp",
		Summary: "Standup",
		Start:   start,
		End:     end,
		RRule:   "FREQ=DAILY;COUNT=5",
	})
	mustContain(t, "recurring", out, "RRULE:FREQ=DAILY;COUNT=5")
}

func TestBuildEvent_Folding(t *testing.T) {
	long := strings.Repeat("A", 200)
	out := BuildEvent(Event{
		UID:     "fold@dav-mcp",
		Summary: long,
		Start:   start,
		End:     end,
	})
	for _, line := range strings.Split(out, "\r\n") {
		if len(line) > 75 {
			t.Errorf("line exceeds 75 chars (%d): %q", len(line), line)
		}
	}
}

func TestBuildEvent_EscapesSpecialChars(t *testing.T) {
	out := BuildEvent(Event{
		UID:         "esc@dav-mcp",
		Summary:     "A, B; C",
		Description: "line1\nline2",
		Start:       start,
		End:         end,
	})
	mustContain(t, "escape", out, `SUMMARY:A\, B\; C`)
	mustContain(t, "escape", out, `DESCRIPTION:line1\nline2`)
}

func TestBuildTodo_Basic(t *testing.T) {
	out := BuildTodo(Todo{
		UID:      "todo-1@dav-mcp",
		Summary:  "Buy milk",
		Due:      due,
		Priority: 1,
	})
	mustContain(t, "todo", out, "BEGIN:VTODO")
	mustContain(t, "todo", out, "UID:todo-1@dav-mcp")
	mustContain(t, "todo", out, "SUMMARY:Buy milk")
	mustContain(t, "todo", out, "DUE:20260430T180000Z")
	mustContain(t, "todo", out, "PRIORITY:1")
	mustContain(t, "todo", out, "END:VTODO")
}

func TestBuildTodo_NoDue(t *testing.T) {
	out := BuildTodo(Todo{UID: "todo-2@dav-mcp", Summary: "No due"})
	mustNotContain(t, "todo-nodue", out, "DUE:")
	mustNotContain(t, "todo-nodue", out, "PRIORITY:")
}

func TestBuildJournal(t *testing.T) {
	out := BuildJournal(Journal{
		UID:         "journal-1@dav-mcp",
		Summary:     "Sprint retrospective",
		Description: "All went well",
		Date:        day,
	})
	mustContain(t, "journal", out, "BEGIN:VJOURNAL")
	mustContain(t, "journal", out, "DTSTART;VALUE=DATE:20260501")
	mustContain(t, "journal", out, "SUMMARY:Sprint retrospective")
	mustContain(t, "journal", out, "DESCRIPTION:All went well")
	mustContain(t, "journal", out, "END:VJOURNAL")
}

func TestNewUID_Unique(t *testing.T) {
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		uid := NewUID()
		if ids[uid] {
			t.Fatalf("duplicate UID: %s", uid)
		}
		ids[uid] = true
		if !strings.HasSuffix(uid, "@dav-mcp") {
			t.Errorf("unexpected UID format: %s", uid)
		}
	}
}

func TestCRLF(t *testing.T) {
	out := BuildEvent(Event{UID: "crlf@dav-mcp", Summary: "X", Start: start, End: end})
	if !strings.Contains(out, "\r\n") {
		t.Error("output must use CRLF line endings")
	}
	if strings.Contains(out, "\n\n") {
		t.Error("bare LF found — CRLF required throughout")
	}
}
