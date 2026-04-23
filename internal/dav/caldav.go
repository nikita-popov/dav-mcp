package dav

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
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
