package dav

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/nikita-popov/dav-mcp/internal/ical"
)

// timeRangeReportTmpl is a CalDAV calendar-query REPORT that fetches all
// VEVENT components in the given time range.
var timeRangeReportTmpl = template.Must(template.New("tr").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT">
        <c:time-range start="{{.Start}}" end="{{.End}}"/>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`))

// uidReportBody is a CalDAV calendar-multiget REPORT body template.
// It fetches the full calendar-data for a known set of hrefs.
var uidReportTmpl = template.Must(template.New("uid").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VEVENT">
        <c:prop-filter name="UID">
          <c:text-match collation="i;unicode-casemap">{{.UID}}</c:text-match>
        </c:prop-filter>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`))

// QueryEvents sends a CalDAV calendar-query REPORT to calendarPath and returns
// the raw calendar-data strings from all matching responses.
// start and end must be in iCalendar basic UTC format: "20060102T150405Z".
func QueryEvents(ctx context.Context, c *Client, calendarPath, start, end string) ([]string, error) {
	var buf bytes.Buffer
	if err := timeRangeReportTmpl.Execute(&buf, struct{ Start, End string }{start, end}); err != nil {
		return nil, fmt.Errorf("caldav: build report: %w", err)
	}

	ms, err := c.Report(ctx, calendarPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("caldav: report: %w", err)
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

// EventRecord pairs a parsed event with the server-side Href and ETag.
type EventRecord struct {
	Event ical.ParsedEvent
	Href  string
	ETag  string
}

// QueryEventByUID searches calendarPath for a VEVENT with the given UID.
// Returns the first match or an error if not found.
func QueryEventByUID(ctx context.Context, c *Client, calendarPath, uid string) (*EventRecord, error) {
	var buf bytes.Buffer
	if err := uidReportTmpl.Execute(&buf, struct{ UID string }{uid}); err != nil {
		return nil, fmt.Errorf("caldav: build uid report: %w", err)
	}
	ms, err := c.Report(ctx, calendarPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("caldav: uid report: %w", err)
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.CalendarData == "" {
				continue
			}
			for _, ev := range ical.ParseEvents(ps.Prop.CalendarData) {
				if ev.UID == uid {
					return &EventRecord{
						Event: ev,
						Href:  r.Href,
						ETag:  ps.Prop.ETag,
					}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("event UID %q not found in %s", uid, calendarPath)
}

// PutEvent stores an iCalendar object at calendarPath/uid.ics.
// If etag is empty the request uses If-None-Match:* (safe create).
// Pass a non-empty etag to update an existing resource.
func PutEvent(ctx context.Context, c *Client, calendarPath, uid, icsData, etag string) error {
	path := calendarPath + uid + ".ics"
	return c.Put(ctx, path, "text/calendar; charset=utf-8", etag, []byte(icsData))
}

// PutEventHref stores an iCalendar object at an explicit server href.
// Used for updates where the exact href is known from a previous REPORT.
func PutEventHref(ctx context.Context, c *Client, href, icsData, etag string) error {
	return c.Put(ctx, href, "text/calendar; charset=utf-8", etag, []byte(icsData))
}

// DeleteEvent removes the calendar resource at calendarPath/uid.ics.
// Pass the ETag obtained from QueryEventByUID for a conditional DELETE
// (prevents deleting a resource that has been modified since it was read).
// Pass an empty etag to skip the If-Match check.
func DeleteEvent(ctx context.Context, c *Client, calendarPath, uid, etag string) error {
	path := calendarPath + uid + ".ics"
	return c.Delete(ctx, path, etag)
}
