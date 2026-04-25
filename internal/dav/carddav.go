package dav

import (
	"context"
	"fmt"

	"github.com/nikita-popov/dav-mcp/internal/vcard"
)

// addressBookReportBody requests all vCard objects in an address book.
var addressBookReportBody = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<c:addressbook-query xmlns:d="DAV:" xmlns:c="urn:ietf:params:xml:ns:carddav">
  <d:prop>
    <d:getetag/>
    <c:address-data/>
  </d:prop>
</c:addressbook-query>`)

// QueryContacts sends a CardDAV addressbook-query REPORT and returns raw
// vCard strings for every contact in the address book at path.
// Deprecated: use QueryContactsFull when Href/ETag are needed.
func QueryContacts(ctx context.Context, c *Client, abPath string) ([]string, error) {
	ms, err := c.Report(ctx, abPath, addressBookReportBody)
	if err != nil {
		return nil, fmt.Errorf("carddav: report: %w", err)
	}
	var out []string
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.AddressData != "" {
				out = append(out, ps.Prop.AddressData)
			}
		}
	}
	return out, nil
}

// ContactRecord pairs a parsed Contact with the server-side Href and ETag
// needed for conditional PUT / DELETE.
type ContactRecord struct {
	Contact vcard.Contact
	Href    string
	ETag    string
}

// QueryContactsFull sends the same REPORT as QueryContacts but returns
// ContactRecord values that include the server Href and ETag per resource.
func QueryContactsFull(ctx context.Context, c *Client, abPath string) ([]ContactRecord, error) {
	ms, err := c.Report(ctx, abPath, addressBookReportBody)
	if err != nil {
		return nil, fmt.Errorf("carddav: report: %w", err)
	}
	var out []ContactRecord
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			if ps.Prop.AddressData == "" {
				continue
			}
			for _, c := range vcard.ParseContacts(ps.Prop.AddressData) {
				out = append(out, ContactRecord{
					Contact: c,
					Href:    r.Href,
					ETag:    ps.Prop.ETag,
				})
			}
		}
	}
	return out, nil
}

// PutContact stores a vCard at abPath/uid.vcf.
func PutContact(ctx context.Context, c *Client, abPath, uid, vcfData, etag string) error {
	path := abPath + uid + ".vcf"
	return c.Put(ctx, path, "text/vcard; charset=utf-8", etag, []byte(vcfData))
}

// PutContactHref stores a vCard at an explicit server href (used for updates
// where the href is known from a previous REPORT).
func PutContactHref(ctx context.Context, c *Client, href, vcfData, etag string) error {
	return c.Put(ctx, href, "text/vcard; charset=utf-8", etag, []byte(vcfData))
}

// DeleteContact removes the vCard resource at abPath/uid.vcf.
// Pass the ETag obtained from QueryContactsFull for a conditional DELETE
// (prevents deleting a resource that has been modified since it was read).
// Pass an empty etag to skip the If-Match check.
func DeleteContact(ctx context.Context, c *Client, abPath, uid, etag string) error {
	path := abPath + uid + ".vcf"
	return c.Delete(ctx, path, etag)
}
