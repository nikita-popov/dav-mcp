package vcard

import (
	"bufio"
	"strings"
)

// ParseContacts parses one or more vCard objects from a single string.
// Multiple BEGIN:VCARD...END:VCARD blocks are all extracted.
// Uses the Contact type and unfold helper defined in builder.go.
func ParseContacts(data string) []Contact {
	var contacts []Contact
	var cur *Contact

	scanner := bufio.NewScanner(strings.NewReader(unfold(data)))
	for scanner.Scan() {
		l := scanner.Text()
		switch {
		case l == "BEGIN:VCARD":
			cur = &Contact{}
		case l == "END:VCARD":
			if cur != nil {
				contacts = append(contacts, *cur)
				cur = nil
			}
		default:
			if cur == nil {
				continue
			}
			propName, value, ok := cutProp(l)
			if !ok {
				continue
			}
			// propName may include params: "EMAIL;TYPE=work" — strip to base name
			base := propName
			if semi := strings.IndexByte(propName, ';'); semi >= 0 {
				base = propName[:semi]
			}
			switch base {
			case "UID":
				cur.UID = value
			case "FN":
				cur.FN = unescape(value)
			case "ORG":
				// ORG may be "Company;Dept" — take first component
				cur.Org = unescape(strings.SplitN(value, ";", 2)[0])
			case "NOTE":
				cur.Notes = unescape(value)
			case "EMAIL":
				// Keep only the first email encountered
				if cur.Email == "" && value != "" {
					cur.Email = value
				}
			case "TEL":
				// Keep only the first phone encountered
				if cur.Phone == "" && value != "" {
					cur.Phone = value
				}
			}
		}
	}
	return contacts
}

// cutProp splits a vCard property line at the first colon.
// Returns the name part (may include params), the value, and ok.
func cutProp(l string) (name, value string, ok bool) {
	colon := strings.IndexByte(l, ':')
	if colon < 0 {
		return
	}
	return l[:colon], l[colon+1:], true
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
