package vcard

import (
	"bufio"
	"strings"
)

// Contact holds the fields we extract from a vCard object.
type Contact struct {
	UID   string
	FN    string // formatted name
	Email []string
	Phone []string
	Org   string
	Note  string
}

// ParseContacts parses one or more vCard objects from a single string.
// Multiple BEGIN:VCARD...END:VCARD blocks are all extracted.
func ParseContacts(data string) []Contact {
	var contacts []Contact
	var cur *Contact

	scanner := bufio.NewScanner(strings.NewReader(unfold(data)))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "BEGIN:VCARD":
			cur = &Contact{}
		case line == "END:VCARD":
			if cur != nil {
				contacts = append(contacts, *cur)
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
			switch {
			case name == "UID":
				cur.UID = value
			case name == "FN":
				cur.FN = unescape(value)
			case name == "ORG":
				cur.Org = unescape(strings.SplitN(value, ";", 2)[0])
			case name == "NOTE":
				cur.Note = unescape(value)
			case name == "EMAIL" || strings.HasPrefix(name, "EMAIL;"):
				if value != "" {
					cur.Email = append(cur.Email, value)
				}
			case name == "TEL" || strings.HasPrefix(name, "TEL;"):
				if value != "" {
					cur.Phone = append(cur.Phone, value)
				}
			}
		}
	}
	return contacts
}

// unfold removes RFC 6350 line folding (CRLF + whitespace).
func unfold(s string) string {
	s = strings.ReplaceAll(s, "\r\n ", "")
	s = strings.ReplaceAll(s, "\r\n\t", "")
	return s
}

// cutProp splits "NAME" or "NAME;params" from value at the first colon.
// Returns the raw name part (including params), the value, and ok.
func cutProp(line string) (name, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon < 0 {
		return
	}
	return line[:colon], line[colon+1:], true
}

// unescape reverses RFC 6350 value escaping.
func unescape(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\N`, "\n")
	s = strings.ReplaceAll(s, `\,`, ",")
	s = strings.ReplaceAll(s, `\;`, ";")
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}
