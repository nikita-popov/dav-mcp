package vcard

import (
	"strings"
	"time"
)

const vcardVersion = "4.0"

// Contact holds the fields for a vCard 4.0 component.
type Contact struct {
	UID   string
	FN    string // formatted/full name (required by RFC 6350)
	Email string
	Phone string
	Org   string
	Notes string
}

// Build produces a vCard 4.0 string for the given contact.
// UID is generated via NewUID() if empty.
func Build(c Contact) string {
	if c.UID == "" {
		c.UID = NewUID()
	}
	var b strings.Builder
	line(&b, "BEGIN", "VCARD")
	line(&b, "VERSION", vcardVersion)
	line(&b, "UID", c.UID)
	line(&b, "FN", escape(c.FN))
	line(&b, "REV", fmtTime(time.Now().UTC()))
	if c.Email != "" {
		line(&b, "EMAIL", c.Email)
	}
	if c.Phone != "" {
		line(&b, "TEL", c.Phone)
	}
	if c.Org != "" {
		line(&b, "ORG", escape(c.Org))
	}
	if c.Notes != "" {
		line(&b, "NOTE", escape(c.Notes))
	}
	line(&b, "END", "VCARD")
	return b.String()
}

// line writes "name:value\r\n" with RFC 6350 §3.2 line folding.
// First line: max 75 octets. Continuation lines: max 74 octets (75 - 1 space).
func line(b *strings.Builder, name, value string) {
	s := name + ":" + value
	const first = 75
	const cont = 74
	if len(s) <= first {
		b.WriteString(s)
		b.WriteString("\r\n")
		return
	}
	b.WriteString(s[:first])
	b.WriteString("\r\n")
	s = s[first:]
	for len(s) > cont {
		b.WriteString(" ")
		b.WriteString(s[:cont])
		b.WriteString("\r\n")
		s = s[cont:]
	}
	b.WriteString(" ")
	b.WriteString(s)
	b.WriteString("\r\n")
}

// fmtTime formats a UTC time per RFC 6350 §4.3.5 (basic format).
func fmtTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// escape applies RFC 6350 §4 TEXT escaping.
func escape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// ParseFN extracts the FN value from a raw vCard string.
func ParseFN(raw string) string {
	return parseField(raw, "FN")
}

// ParseUID extracts the UID value from a raw vCard string.
func ParseUID(raw string) string {
	return parseField(raw, "UID")
}

// parseField finds the first "NAME:value" line (case-insensitive name)
// and returns the unfolded value. Returns "" if not found.
func parseField(raw, name string) string {
	unfolded := unfold(raw)
	upper := strings.ToUpper(name) + ":"
	for _, l := range strings.Split(unfolded, "\n") {
		l = strings.TrimRight(l, "\r")
		up := strings.ToUpper(l)
		if strings.HasPrefix(up, upper) {
			return l[len(name)+1:]
		}
	}
	return ""
}

// unfold reverses RFC 6350 §3.2 line folding (CRLF + space/tab → nothing).
func unfold(s string) string {
	s = strings.ReplaceAll(s, "\r\n ", "")
	s = strings.ReplaceAll(s, "\r\n\t", "")
	return s
}
