package dav

import (
	"context"
	"fmt"
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

// PutContact stores a vCard at abPath/uid.vcf.
func PutContact(ctx context.Context, c *Client, abPath, uid, vcfData, etag string) error {
	path := abPath + uid + ".vcf"
	return c.Put(ctx, path, "text/vcard; charset=utf-8", etag, []byte(vcfData))
}

// DeleteContact removes the .vcf resource identified by uid from abPath.
func DeleteContact(ctx context.Context, c *Client, abPath, uid string) error {
	path := abPath + uid + ".vcf"
	return c.Delete(ctx, path, "")
}
