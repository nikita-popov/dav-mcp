package ical

import (
	"bufio"
	"strconv"
	"strings"
	"time"
)

// ParsedEvent holds fields extracted from a VEVENT block.
type ParsedEvent struct {
	UID          string
	Summary      string
	Description  string
	Location     string
	Start        time.Time
	End          time.Time
	StartTZ      string // raw TZID from DTSTART, empty for UTC/floating
	EndTZ        string // raw TZID from DTEND, empty for UTC/floating
	RRule        string
	RecurrenceID string // set on expanded occurrences (server adds after <c:expand>)
	Sequence     int
}

// ParsedTodo holds fields extracted from a VTODO block.
type ParsedTodo struct {
	UID         string
	Summary     string
	Description string
	Due         time.Time // zero = no due date
	Priority    int       // 0 = undefined, 1-9 per RFC 5545
	Status      string    // e.g. "NEEDS-ACTION", "COMPLETED", "IN-PROCESS"
}

// ParsedJournal holds fields extracted from a VJOURNAL block.
type ParsedJournal struct {
	UID         string
	Summary     string
	Description string
	Date        time.Time
	Status      string // e.g. "DRAFT", "FINAL", "CANCELLED"
}

// ParseEvents extracts all VEVENT blocks from an iCalendar string.
// Unrecognised or malformed properties are silently skipped.
func ParseEvents(data string) []ParsedEvent {
	var events []ParsedEvent
	var cur *ParsedEvent

	scanner := bufio.NewScanner(strings.NewReader(unfold(data)))
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "BEGIN:VEVENT":
			cur = &ParsedEvent{}
		case "END:VEVENT":
			if cur != nil {
				events = append(events, *cur)
				cur = nil
			}
		default:
			if cur == nil {
				continue
			}
			name, params, value, ok := cutPropFull(line)
			if !ok {
				continue
			}
			switch name {
			case "UID":
				cur.UID = value
			case "SUMMARY":
				cur.Summary = unescape(value)
			case "DESCRIPTION":
				cur.Description = unescape(value)
			case "LOCATION":
				cur.Location = unescape(value)
			case "RRULE":
				cur.RRule = value
			case "RECURRENCE-ID":
				cur.RecurrenceID = value
			case "SEQUENCE":
				if n, err := strconv.Atoi(value); err == nil {
					cur.Sequence = n
				}
			case "DTSTART":
				tzid := extractTZID(params)
				cur.StartTZ = tzid
				cur.Start = parseTimeWithTZ(value, tzid)
			case "DTEND":
				tzid := extractTZID(params)
				cur.EndTZ = tzid
				cur.End = parseTimeWithTZ(value, tzid)
			}
		}
	}
	return events
}

// ParseTodos extracts all VTODO blocks from an iCalendar string.
func ParseTodos(data string) []ParsedTodo {
	var todos []ParsedTodo
	var cur *ParsedTodo

	scanner := bufio.NewScanner(strings.NewReader(unfold(data)))
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "BEGIN:VTODO":
			cur = &ParsedTodo{}
		case "END:VTODO":
			if cur != nil {
				todos = append(todos, *cur)
				cur = nil
			}
		default:
			if cur == nil {
				continue
			}
			name, params, value, ok := cutPropFull(line)
			if !ok {
				continue
			}
			switch name {
			case "UID":
				cur.UID = value
			case "SUMMARY":
				cur.Summary = unescape(value)
			case "DESCRIPTION":
				cur.Description = unescape(value)
			case "DUE":
				cur.Due = parseTimeWithTZ(value, extractTZID(params))
			case "PRIORITY":
				if n, err := strconv.Atoi(value); err == nil {
					cur.Priority = n
				}
			case "STATUS":
				cur.Status = value
			}
		}
	}
	return todos
}

// ParseJournals extracts all VJOURNAL blocks from an iCalendar string.
func ParseJournals(data string) []ParsedJournal {
	var journals []ParsedJournal
	var cur *ParsedJournal

	scanner := bufio.NewScanner(strings.NewReader(unfold(data)))
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "BEGIN:VJOURNAL":
			cur = &ParsedJournal{}
		case "END:VJOURNAL":
			if cur != nil {
				journals = append(journals, *cur)
				cur = nil
			}
		default:
			if cur == nil {
				continue
			}
			name, params, value, ok := cutPropFull(line)
			if !ok {
				continue
			}
			switch name {
			case "UID":
				cur.UID = value
			case "SUMMARY":
				cur.Summary = unescape(value)
			case "DESCRIPTION":
				cur.Description = unescape(value)
			case "DTSTART":
				cur.Date = parseTimeWithTZ(value, extractTZID(params))
			case "STATUS":
				cur.Status = value
			}
		}
	}
	return journals
}

// unfold removes RFC 5545 line folding (CRLF + whitespace continuation).
func unfold(s string) string {
	s = strings.ReplaceAll(s, "\r\n ", "")
	s = strings.ReplaceAll(s, "\r\n\t", "")
	return s
}

// cutPropFull splits "NAME;param1=v1;param2=v2:value" into
// (name, params-string, value). Returns ok=false if no colon found.
func cutPropFull(line string) (name, params, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return
	}
	namepart := line[:colon]
	value = line[colon+1:]
	if semi := strings.IndexByte(namepart, ';'); semi >= 0 {
		name = namepart[:semi]
		params = namepart[semi+1:]
	} else {
		name = namepart
	}
	return name, params, value, true
}

// cutProp is the legacy helper kept for callers that don't need params.
// Deprecated: prefer cutPropFull.
func cutProp(line string) (name, value string, ok bool) {
	n, _, v, o := cutPropFull(line)
	return n, v, o
}

// extractTZID returns the TZID value from a params string like
// "TZID=Asia/Yekaterinburg" or "TZID=Europe/Moscow;VALUE=DATE-TIME".
// Returns empty string if not present.
func extractTZID(params string) string {
	for _, p := range strings.Split(params, ";") {
		if strings.HasPrefix(p, "TZID=") {
			return p[5:]
		}
	}
	return ""
}

// parseTimeWithTZ parses a DTSTART/DTEND value.
// tzid is the raw TZID parameter value (may be empty).
// For UTC values (ending in Z) the timezone is always UTC.
// For floating values (no Z, no tzid) UTC is assumed.
// For localised values the Go time package is used to load the named location;
// if the location is unknown the value is parsed as floating UTC.
func parseTimeWithTZ(value, tzid string) time.Time {
	// All-day date: no time component
	if len(value) == 8 {
		t, _ := time.Parse("20060102", value)
		return t
	}
	// UTC explicit
	if strings.HasSuffix(value, "Z") {
		t, _ := time.Parse("20060102T150405Z", value)
		return t
	}
	// Localised with TZID
	if tzid != "" {
		if loc, err := time.LoadLocation(tzid); err == nil {
			t, err := time.ParseInLocation("20060102T150405", value, loc)
			if err == nil {
				return t
			}
		}
	}
	// Floating: treat as UTC
	t, _ := time.Parse("20060102T150405", value)
	return t
}

// parseTime is the legacy single-arg wrapper.
func parseTime(prop, value string) time.Time {
	var tzid string
	if strings.Contains(prop, "TZID=") {
		if i := strings.Index(prop, "TZID="); i >= 0 {
			tzid = prop[i+5:]
			if j := strings.IndexByte(tzid, ';'); j >= 0 {
				tzid = tzid[:j]
			}
		}
	}
	return parseTimeWithTZ(value, tzid)
}

// unescape reverses RFC 5545 §3.3.11 TEXT escaping.
func unescape(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\N`, "\n")
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\;`, ";")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// IsRecurring reports whether the event is part of a recurring series.
// True for master events (have RRULE) and expanded occurrences (have RECURRENCE-ID).
func (e *ParsedEvent) IsRecurring() bool {
	return e.RRule != "" || e.RecurrenceID != ""
}
