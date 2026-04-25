package ical

import (
	"fmt"
	"strings"
	"time"
)

const (
	timeFormat  = "20060102T150405Z"
	dateFormat  = "20060102"
	prodID      = "-//dav-mcp//EN"
	icalVersion = "2.0"
)

// Event holds the fields for a VEVENT component.
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool   // if true, use DATE format for DTSTART/DTEND
	RRule       string // raw RRULE value, e.g. "FREQ=WEEKLY;BYDAY=MO,WE"
	Sequence    int    // incremented on each update per RFC 5545 §3.8.7.4
}

// Todo holds the fields for a VTODO component.
type Todo struct {
	UID         string
	Summary     string
	Description string
	Due         time.Time // zero = no due date
	Priority    int       // 0 = undefined, 1-9 per RFC 5545
}

// Journal holds the fields for a VJOURNAL component.
type Journal struct {
	UID         string
	Summary     string
	Description string
	Date        time.Time
}

// BuildEvent produces a VCALENDAR/VEVENT iCalendar string.
func BuildEvent(e Event) string {
	if e.UID == "" {
		e.UID = NewUID()
	}
	now := fmtTime(time.Now().UTC())
	var b strings.Builder
	prop(&b, "BEGIN", "VCALENDAR")
	prop(&b, "VERSION", icalVersion)
	prop(&b, "PRODID", prodID)
	prop(&b, "BEGIN", "VEVENT")
	prop(&b, "UID", e.UID)
	prop(&b, "DTSTAMP", now)
	prop(&b, "LAST-MODIFIED", now)
	if e.Sequence > 0 {
		prop(&b, "SEQUENCE", fmt.Sprintf("%d", e.Sequence))
	}
	if e.AllDay {
		prop(&b, "DTSTART;VALUE=DATE", e.Start.UTC().Format(dateFormat))
		prop(&b, "DTEND;VALUE=DATE", e.End.UTC().Format(dateFormat))
	} else {
		prop(&b, "DTSTART", fmtTime(e.Start.UTC()))
		prop(&b, "DTEND", fmtTime(e.End.UTC()))
	}
	prop(&b, "SUMMARY", escape(e.Summary))
	if e.Description != "" {
		prop(&b, "DESCRIPTION", escape(e.Description))
	}
	if e.Location != "" {
		prop(&b, "LOCATION", escape(e.Location))
	}
	if e.RRule != "" {
		prop(&b, "RRULE", e.RRule)
	}
	prop(&b, "END", "VEVENT")
	prop(&b, "END", "VCALENDAR")
	return b.String()
}

// BuildTodo produces a VCALENDAR/VTODO iCalendar string.
func BuildTodo(t Todo) string {
	if t.UID == "" {
		t.UID = NewUID()
	}
	now := fmtTime(time.Now().UTC())
	var b strings.Builder
	prop(&b, "BEGIN", "VCALENDAR")
	prop(&b, "VERSION", icalVersion)
	prop(&b, "PRODID", prodID)
	prop(&b, "BEGIN", "VTODO")
	prop(&b, "UID", t.UID)
	prop(&b, "DTSTAMP", now)
	prop(&b, "SUMMARY", escape(t.Summary))
	if !t.Due.IsZero() {
		prop(&b, "DUE", fmtTime(t.Due.UTC()))
	}
	if t.Priority > 0 {
		prop(&b, "PRIORITY", fmt.Sprintf("%d", t.Priority))
	}
	if t.Description != "" {
		prop(&b, "DESCRIPTION", escape(t.Description))
	}
	prop(&b, "END", "VTODO")
	prop(&b, "END", "VCALENDAR")
	return b.String()
}

// BuildJournal produces a VCALENDAR/VJOURNAL iCalendar string.
func BuildJournal(j Journal) string {
	if j.UID == "" {
		j.UID = NewUID()
	}
	now := fmtTime(time.Now().UTC())
	var b strings.Builder
	prop(&b, "BEGIN", "VCALENDAR")
	prop(&b, "VERSION", icalVersion)
	prop(&b, "PRODID", prodID)
	prop(&b, "BEGIN", "VJOURNAL")
	prop(&b, "UID", j.UID)
	prop(&b, "DTSTAMP", now)
	prop(&b, "DTSTART;VALUE=DATE", j.Date.UTC().Format(dateFormat))
	prop(&b, "SUMMARY", escape(j.Summary))
	if j.Description != "" {
		prop(&b, "DESCRIPTION", escape(j.Description))
	}
	prop(&b, "END", "VJOURNAL")
	prop(&b, "END", "VCALENDAR")
	return b.String()
}

// prop writes "name:value\r\n" with RFC 5545 §3.1 line folding.
// First line: max 75 octets. Continuation lines start with a space,
// so usable content per continuation line is 74 octets (75 - 1 space).
func prop(b *strings.Builder, name, value string) {
	line := name + ":" + value
	const firstMax = 75
	const contMax = 74
	if len(line) <= firstMax {
		b.WriteString(line)
		b.WriteString("\r\n")
		return
	}
	b.WriteString(line[:firstMax])
	b.WriteString("\r\n")
	line = line[firstMax:]
	for len(line) > contMax {
		b.WriteString(" ")
		b.WriteString(line[:contMax])
		b.WriteString("\r\n")
		line = line[contMax:]
	}
	b.WriteString(" ")
	b.WriteString(line)
	b.WriteString("\r\n")
}

// fmtTime formats a UTC time in iCalendar basic format.
func fmtTime(t time.Time) string {
	return t.UTC().Format(timeFormat)
}

// escape applies RFC 5545 §3.3.11 TEXT escaping.
func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}
