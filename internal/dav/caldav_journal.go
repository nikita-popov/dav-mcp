package dav

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/nikita-popov/dav-mcp/internal/ical"
)

var journalAllTmpl = template.Must(template.New("ja").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VJOURNAL"/>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`))

var journalUIDTmpl = template.Must(template.New("ju").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VJOURNAL">
        <c:prop-filter name="UID">
          <c:text-match collation="i;unicode-casemap">{{.UID}}</c:text-match>
        </c:prop-filter>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`))

// JournalRecord pairs a parsed journal with the server-side Href and ETag.
type JournalRecord struct {
	Journal ical.ParsedJournal
	Href    string
	ETag    string
}

// QueryJournals fetches all VJOURNAL components from calendarPath.
func QueryJournals(ctx context.Context, c *Client, calendarPath string) ([]string, error) {
	var buf bytes.Buffer
	if err := journalAllTmpl.Execute(&buf, nil); err != nil {
		return nil, fmt.Errorf("caldav: build journal report: %w", err)
	}
	ms, err := c.Report(ctx, calendarPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("caldav: journal report: %w", err)
	}
	var out []string
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.CalendarData != "" {
				out = append(out, ps.Prop.CalendarData)
			}
		}
	}
	return out, nil
}

// QueryJournalByUID searches calendarPath for a VJOURNAL with the given UID.
func QueryJournalByUID(ctx context.Context, c *Client, calendarPath, uid string) (*JournalRecord, error) {
	var buf bytes.Buffer
	if err := journalUIDTmpl.Execute(&buf, struct{ UID string }{uid}); err != nil {
		return nil, fmt.Errorf("caldav: build journal uid report: %w", err)
	}
	ms, err := c.Report(ctx, calendarPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("caldav: journal uid report: %w", err)
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.CalendarData == "" {
				continue
			}
			for _, j := range ical.ParseJournals(ps.Prop.CalendarData) {
				if j.UID == uid {
					return &JournalRecord{
						Journal: j,
						Href:    r.Href,
						ETag:    ps.Prop.ETag,
					}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("journal UID %q not found in %s", uid, calendarPath)
}

// PutJournal stores a VJOURNAL iCalendar object at calendarPath/uid.ics.
// Pass empty etag for creation (If-None-Match:*), non-empty for update.
func PutJournal(ctx context.Context, c *Client, calendarPath, uid, icsData, etag string) error {
	path := calendarPath + uid + ".ics"
	return c.Put(ctx, path, "text/calendar; charset=utf-8", etag, []byte(icsData))
}

// PutJournalHref stores a VJOURNAL at an explicit server href (used for updates).
func PutJournalHref(ctx context.Context, c *Client, href, icsData, etag string) error {
	return c.Put(ctx, href, "text/calendar; charset=utf-8", etag, []byte(icsData))
}

// DeleteJournal removes the .ics resource at href.
func DeleteJournal(ctx context.Context, c *Client, href, etag string) error {
	return c.Delete(ctx, href, etag)
}
