package ical

import (
	"strconv"
	"strings"
	"time"
)

// ExpandEvents replaces each recurring master event (has RRULE, no RECURRENCE-ID)
// with its individual occurrences within [winStart, winEnd).
// Non-recurring events and already-expanded occurrences pass through unchanged.
func ExpandEvents(events []ParsedEvent, winStart, winEnd time.Time) []ParsedEvent {
	out := make([]ParsedEvent, 0, len(events))
	for _, ev := range events {
		if ev.RRule != "" && ev.RecurrenceID == "" {
			out = append(out, ExpandRecurring(ev, winStart, winEnd)...)
		} else {
			out = append(out, ev)
		}
	}
	return out
}

// ExpandRecurring generates all occurrences of a recurring event within
// the half-open window [winStart, winEnd).
// Each returned event has RecurrenceID set to the occurrence DTSTART value
// (RFC 5545 §3.8.4.4) and RRule cleared.
func ExpandRecurring(ev ParsedEvent, winStart, winEnd time.Time) []ParsedEvent {
	rule := parseRRule(ev.RRule)
	dur := ev.End.Sub(ev.Start)

	var results []ParsedEvent
	count := 0

	for cur := ev.Start; ; cur = rule.next(cur) {
		// Hard limit to avoid infinite loops on malformed rules.
		if count > 3650 {
			break
		}
		// UNTIL boundary
		if !rule.until.IsZero() && cur.After(rule.until) {
			break
		}
		// COUNT boundary
		if rule.count > 0 && count >= rule.count {
			break
		}
		// Past the window — stop (occurrences are monotonically increasing)
		if !cur.Before(winEnd) {
			break
		}
		count++
		// Skip occurrences before the window
		if cur.Before(winStart) {
			continue
		}
		occ := ev
		occ.Start = cur
		occ.End = cur.Add(dur)
		occ.RecurrenceID = cur.UTC().Format("20060102T150405Z")
		occ.RRule = ""
		results = append(results, occ)
	}
	return results
}

// rrule holds the parsed subset of RRULE we support.
type rrule struct {
	freq     string    // DAILY WEEKLY MONTHLY YEARLY
	interval int       // default 1
	count    int       // 0 = unlimited
	until    time.Time // zero = unlimited
	byday    []string  // e.g. ["MO","TU"] for WEEKLY
}

var weekdayIndex = map[string]time.Weekday{
	"SU": time.Sunday,
	"MO": time.Monday,
	"TU": time.Tuesday,
	"WE": time.Wednesday,
	"TH": time.Thursday,
	"FR": time.Friday,
	"SA": time.Saturday,
}

func parseRRule(s string) rrule {
	r := rrule{interval: 1}
	for _, part := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch k {
		case "FREQ":
			r.freq = v
		case "INTERVAL":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				r.interval = n
			}
		case "COUNT":
			if n, err := strconv.Atoi(v); err == nil {
				r.count = n
			}
		case "UNTIL":
			r.until = parseTimeWithTZ(v, "")
		case "BYDAY":
			for _, day := range strings.Split(v, ",") {
				// Strip optional ordinal prefix (+1MO → MO)
				day = strings.TrimLeft(day, "+-0123456789")
				if len(day) == 2 {
					r.byday = append(r.byday, day)
				}
			}
		}
	}
	return r
}

// next returns the next candidate occurrence after cur.
// For WEEKLY+BYDAY it advances day-by-day within the week set,
// for other frequencies it advances by the full interval.
func (r rrule) next(cur time.Time) time.Time {
	if r.freq == "WEEKLY" && len(r.byday) > 0 {
		// Advance one day at a time; jump a week when we've exhausted all BYDAY
		// entries in the current week.
		next := cur.AddDate(0, 0, 1)
		// Find the nearest BYDAY at or after next within the same ISO week.
		for i := 0; i < 7*r.interval; i++ {
			wd := next.Weekday()
			for _, d := range r.byday {
				if weekdayIndex[d] == wd {
					return next
				}
			}
			next = next.AddDate(0, 0, 1)
		}
		// Fallback: plain weekly step
		return cur.AddDate(0, 0, 7*r.interval)
	}
	switch r.freq {
	case "DAILY":
		return cur.AddDate(0, 0, r.interval)
	case "WEEKLY":
		return cur.AddDate(0, 0, 7*r.interval)
	case "MONTHLY":
		return cur.AddDate(0, r.interval, 0)
	case "YEARLY":
		return cur.AddDate(r.interval, 0, 0)
	default:
		// Unknown frequency — step by 1 day to avoid infinite loop
		return cur.AddDate(0, 0, 1)
	}
}
