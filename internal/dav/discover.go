package dav

import (
	"context"
	"encoding/xml"
	"strings"
)

type Principal struct {
	Href string
}

type HomeSet struct {
	Href string
}

type Collection struct {
	Href        string
	DisplayName string
}

type DiscoverProp struct {
	CurrentUserPrincipal struct {
		Href string `xml:"href"`
	} `xml:"current-user-principal"`

	CalendarHomeSet struct {
		Href string `xml:"href"`
	} `xml:"calendar-home-set"`

	AddressbookHomeSet struct {
		Href string `xml:"href"`
	} `xml:"addressbook-home-set"`

	DisplayName string `xml:"displayname"`

	ResourceType struct {
		Collection bool `xml:"collection"`
	} `xml:"resourcetype"`
}

func propfindValue(
	ctx context.Context,
	c *Client,
	path string,
	body []byte,
	extract func(Prop) string,
) (string, error) {
	ms, err := c.Propfind(ctx, path, "0", body)
	if err != nil {
		return "", err
	}
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			v := extract(ps.Prop)
			if v != "" {
				return v, nil
			}
		}
	}
	return "", ErrNotFound
}

func DiscoverPrincipal(ctx context.Context, c *Client) (*Principal, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:">
  <prop>
    <current-user-principal/>
  </prop>
</propfind>`)
	href, err := propfindValue(ctx, c, "/", body, func(p Prop) string {
		return p.CurrentUserPrincipal.Href
	})
	if err != nil {
		return nil, err
	}
	return &Principal{Href: href}, nil
}

func DiscoverCalendarHome(ctx context.Context, c *Client, principal string) (*HomeSet, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:" xmlns:c="urn:ietf:params:xml:ns:caldav">
  <prop>
    <c:calendar-home-set/>
  </prop>
</propfind>`)
	href, err := propfindValue(ctx, c, principal, body, func(p Prop) string {
		return p.CalendarHomeSet.Href
	})
	if err != nil {
		return nil, err
	}
	return &HomeSet{Href: href}, nil
}

func DiscoverAddressbookHome(ctx context.Context, c *Client, principal string) (*HomeSet, error) {
	body := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<propfind xmlns="DAV:" xmlns:card="urn:ietf:params:xml:ns:carddav">
  <prop>
    <card:addressbook-home-set/>
  </prop>
</propfind>`)
	href, err := propfindValue(ctx, c, principal, body, func(p Prop) string {
		return p.AddressbookHomeSet.Href
	})
	if err != nil {
		return nil, err
	}
	return &HomeSet{Href: href}, nil
}

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
		if r.Href == path {
			continue
		}
		for _, ps := range r.Propstat {
			if !ps.Prop.ResourceType.Collection {
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

func init() {
	_ = xml.Name{}
}
