package dav

import (
	"context"
	"strings"
)

// Collection represents a discovered CalDAV calendar or CardDAV address book.
type Collection struct {
	Href        string
	DisplayName string
	// Components lists supported iCalendar component types (VEVENT, VTODO, …).
	// Empty means the server did not advertise supported-calendar-component-set.
	Components []string
}

// Supports reports whether the collection advertises support for the given
// component type (case-insensitive). Returns true when Components is empty
// (server did not advertise anything — assume all are supported).
func (col Collection) Supports(comp string) bool {
	if len(col.Components) == 0 {
		return true
	}
	upper := strings.ToUpper(comp)
	for _, c := range col.Components {
		if strings.ToUpper(c) == upper {
			return true
		}
	}
	return false
}

// DiscoverPrincipal fetches the current-user-principal URL from the server root.
func DiscoverPrincipal(ctx context.Context, c *Client) (string, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:">
  <prop><current-user-principal/></prop>
</propfind>`)
	ms, err := c.Propfind(ctx, "/", "0", body)
	if err != nil {
		return "", err
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if h := ps.Prop.CurrentUserPrincipal.Href; h != "" {
				return h, nil
			}
		}
	}
	return "", ErrNotFound
}

// DiscoverCalendarHome fetches the calendar-home-set URL for the given principal.
func DiscoverCalendarHome(ctx context.Context, c *Client, principal string) (string, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <prop><c:calendar-home-set/></prop>
</propfind>`)
	ms, err := c.Propfind(ctx, principal, "0", body)
	if err != nil {
		return "", err
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if h := ps.Prop.CalendarHomeSet.Href; h != "" {
				return h, nil
			}
		}
	}
	return "", ErrNotFound
}

// DiscoverAddressbookHome fetches the addressbook-home-set URL for the given principal.
func DiscoverAddressbookHome(ctx context.Context, c *Client, principal string) (string, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
  <prop><card:addressbook-home-set/></prop>
</propfind>`)
	ms, err := c.Propfind(ctx, principal, "0", body)
	if err != nil {
		return "", err
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if h := ps.Prop.AddressbookHomeSet.Href; h != "" {
				return h, nil
			}
		}
	}
	return "", ErrNotFound
}

// DiscoverCollections lists child collections under path (depth:1).
// For each calendar collection it also fetches supported-calendar-component-set.
func DiscoverCollections(ctx context.Context, c *Client, path string) ([]Collection, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <prop>
    <displayname/>
    <resourcetype/>
    <c:supported-calendar-component-set/>
  </prop>
</propfind>`)
	ms, err := c.Propfind(ctx, path, "1", body)
	if err != nil {
		return nil, err
	}
	var out []Collection
	for _, r := range ms.Responses {
		if strings.TrimRight(r.Href, "/") == strings.TrimRight(path, "/") {
			continue
		}
		for _, ps := range r.Propstat {
			if !ps.Prop.ResourceType.IsCollection() {
				continue
			}
			out = append(out, Collection{
				Href:        normalizeHref(r.Href),
				DisplayName: ps.Prop.DisplayName,
				Components:  ps.Prop.SupportedCalendarComponentSet.Names(),
			})
		}
	}
	return out, nil
}

func normalizeHref(h string) string {
	if strings.HasSuffix(h, "/") {
		return h
	}
	return h + "/"
}
