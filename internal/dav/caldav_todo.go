package dav

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/nikita-popov/dav-mcp/internal/ical"
)

var todoRangeTmpl = template.Must(template.New("tr").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VTODO"/>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`))

var todoUIDTmpl = template.Must(template.New("uid").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<c:calendar-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <d:prop>
    <d:getetag/>
    <c:calendar-data/>
  </d:prop>
  <c:filter>
    <c:comp-filter name="VCALENDAR">
      <c:comp-filter name="VTODO">
        <c:prop-filter name="UID">
          <c:text-match collation="i;unicode-casemap">{{.UID}}</c:text-match>
        </c:prop-filter>
      </c:comp-filter>
    </c:comp-filter>
  </c:filter>
</c:calendar-query>`))

// TodoRecord pairs a parsed todo with the server-side Href and ETag.
type TodoRecord struct {
	Todo ical.ParsedTodo
	Href string
	ETag string
}

// QueryTodos fetches all VTODO components from calendarPath.
func QueryTodos(ctx context.Context, c *Client, calendarPath string) ([]string, error) {
	var buf bytes.Buffer
	if err := todoRangeTmpl.Execute(&buf, nil); err != nil {
		return nil, fmt.Errorf("caldav: build todo report: %w", err)
	}
	ms, err := c.Report(ctx, calendarPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("caldav: todo report: %w", err)
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

// QueryTodoByUID searches calendarPath for a VTODO with the given UID.
func QueryTodoByUID(ctx context.Context, c *Client, calendarPath, uid string) (*TodoRecord, error) {
	var buf bytes.Buffer
	if err := todoUIDTmpl.Execute(&buf, struct{ UID string }{uid}); err != nil {
		return nil, fmt.Errorf("caldav: build todo uid report: %w", err)
	}
	ms, err := c.Report(ctx, calendarPath, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("caldav: todo uid report: %w", err)
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.CalendarData == "" {
				continue
			}
			for _, t := range ical.ParseTodos(ps.Prop.CalendarData) {
				if t.UID == uid {
					return &TodoRecord{
						Todo: t,
						Href: r.Href,
						ETag: ps.Prop.ETag,
					}, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("todo UID %q not found in %s", uid, calendarPath)
}

// PutTodo stores a VTODO iCalendar object at calendarPath/uid.ics.
// Pass empty etag for creation (If-None-Match:*), non-empty for update.
func PutTodo(ctx context.Context, c *Client, calendarPath, uid, icsData, etag string) error {
	path := calendarPath + uid + ".ics"
	return c.Put(ctx, path, "text/calendar; charset=utf-8", etag, []byte(icsData))
}

// PutTodoHref stores a VTODO at an explicit server href (used for updates).
func PutTodoHref(ctx context.Context, c *Client, href, icsData, etag string) error {
	return c.Put(ctx, href, "text/calendar; charset=utf-8", etag, []byte(icsData))
}

// DeleteTodo removes the .ics resource at href.
func DeleteTodo(ctx context.Context, c *Client, href, etag string) error {
	return c.Delete(ctx, href, etag)
}
