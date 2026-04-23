package dav

import (
	"context"
	"strings"
)

// Collection represents a discovered CalDAV calendar or CardDAV address book.
type Collection struct {
	Href        string
	DisplayName string
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
// Skips the path itself and non-collection resources.
func DiscoverCollections(ctx context.Context, c *Client, path string) ([]Collection, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:">
  <prop>
    <displayname/>
    <resourcetype/>
  </prop>
</propfind>`)
	ms, err := c.Propfind(ctx, path, "1", body)
	if err != nil {
		return nil, err
	}
	var out []Collection
	for _, r := range ms.Responses {
		// skip the collection itself
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
