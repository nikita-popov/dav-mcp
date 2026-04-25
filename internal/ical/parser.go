package ical

import (
	"bufio"
	"strconv"
	"strings"
	"time"
)

// ParsedEvent holds fields extracted from a VEVENT block.
type ParsedEvent struct {
	UID         string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	RRule       string
	Sequence    int
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
			name, value, ok := cutProp(line)
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
			case "SEQUENCE":
				if n, err := strconv.Atoi(value); err == nil {
					cur.Sequence = n
				}
			case "DTSTART", "DTSTART;TZID", "DTSTART;VALUE=DATE":
				cur.Start = parseTime(name, value)
			case "DTEND", "DTEND;TZID", "DTEND;VALUE=DATE":
				cur.End = parseTime(name, value)
			}
		}
	}
	return events
}

// unfold removes RFC 5545 line folding (CRLF + whitespace continuation).
func unfold(s string) string {
	s = strings.ReplaceAll(s, "\r\n ", "")
	s = strings.ReplaceAll(s, "\r\n\t", "")
	return s
}

// cutProp splits "NAME;params:value" into ("NAME", "value").
// Params (e.g. TZID=...) are stripped from the name part for matching.
func cutProp(line string) (name, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return
	}
	namepart := line[:colon]
	value = line[colon+1:]
	// strip param: "DTSTART;TZID=America/New_York" → "DTSTART"
	// but keep "DTSTART;VALUE=DATE" as-is for parseTime to detect all-day
	if semi := strings.IndexByte(namepart, ';'); semi >= 0 {
		base := namepart[:semi]
		param := namepart[semi+1:]
		if strings.HasPrefix(param, "VALUE=DATE") {
			name = base + ";VALUE=DATE"
		} else {
			name = base
		}
	} else {
		name = namepart
	}
	return name, value, true
}

// parseTime parses DTSTART / DTEND values.
func parseTime(prop, value string) time.Time {
	if strings.HasSuffix(prop, "VALUE=DATE") {
		t, _ := time.Parse("20060102", value)
		return t
	}
	// UTC: ends with Z
	if strings.HasSuffix(value, "Z") {
		t, _ := time.Parse("20060102T150405Z", value)
		return t
	}
	// floating (no Z, no TZID) — parse as UTC
	t, _ := time.Parse("20060102T150405", value)
	return t
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
