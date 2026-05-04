package dav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

type Client struct {
	BaseURL  string
	HTTP     *http.Client
	Username string
	Password string
}

func New(base string, username string, password string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("DAV URL must use http or https, got %q", u.Scheme)
	}
	return &Client{
		BaseURL:  base,
		Username: username,
		Password: password,
		HTTP:     &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) resolve(path string) string {
	if len(path) >= 7 && (path[:7] == "http://" || (len(path) >= 8 && path[:8] == "https://")) {
		return path
	}
	base, _ := url.Parse(c.BaseURL)
	ref, _ := url.Parse(path)
	return base.ResolveReference(ref).String()
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
	body io.Reader,
) (*http.Response, error) {
	target := c.resolve(path)
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	req.Header.Set("User-Agent", "dav-mcp")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	mcp.Debugf("HTTP %s %s", method, target)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	mcp.Debugf("HTTP %d %s", resp.StatusCode, target)
	return resp, nil
}

func (c *Client) Propfind(
	ctx context.Context,
	path string,
	depth string,
	body []byte,
) (*MultiStatus, error) {
	headers := map[string]string{
		"Depth":        depth,
		"Content-Type": "application/xml",
	}
	resp, err := c.do(ctx, "PROPFIND", path, headers, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 207 {
		return nil, mapHTTPError(resp.StatusCode)
	}
	var ms MultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, err
	}
	return &ms, nil
}

// Report sends a CalDAV REPORT request.
// RFC 4791 §7.8 does not restrict calendar-query to Depth:1;
// using "infinity" ensures servers like Yandex return all matching objects.
func (c *Client) Report(
	ctx context.Context,
	path string,
	body []byte,
) (*MultiStatus, error) {
	headers := map[string]string{
		"Depth":        "infinity",
		"Content-Type": "application/xml",
	}
	resp, err := c.do(ctx, "REPORT", path, headers, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 207 {
		return nil, mapHTTPError(resp.StatusCode)
	}
	var ms MultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, err
	}
	return &ms, nil
}

func (c *Client) Put(
	ctx context.Context,
	path string,
	contentType string,
	etag string,
	body []byte,
) error {
	headers := map[string]string{"Content-Type": contentType}
	if etag == "" {
		headers["If-None-Match"] = "*"
	} else {
		headers["If-Match"] = etag
	}
	resp, err := c.do(ctx, "PUT", path, headers, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return mapHTTPError(resp.StatusCode)
}

func (c *Client) Delete(ctx context.Context, path string, etag string) error {
	headers := map[string]string{}
	if etag != "" {
		headers["If-Match"] = etag
	}
	resp, err := c.do(ctx, "DELETE", path, headers, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return mapHTTPError(resp.StatusCode)
}

func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, mapHTTPError(resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// MultiStatus represents a WebDAV 207 Multi-Status response.
type MultiStatus struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []Response `xml:"response"`
}

type Response struct {
	Href     string     `xml:"href"`
	Propstat []Propstat `xml:"propstat"`
}

type Propstat struct {
	Status string `xml:"status"`
	Prop   Prop   `xml:"prop"`
}

// Prop holds WebDAV/CalDAV/CardDAV properties from a PROPFIND or REPORT response.
// Each CalDAV/CardDAV field has two struct tags: the namespaced form (used by
// strict servers such as Yandex and Nextcloud) and a plain fallback (used by
// servers that omit the namespace prefix in the response body).
type Prop struct {
	DisplayName string `xml:"displayname,omitempty"`
	ETag        string `xml:"getetag,omitempty"`

	// CalDAV calendar-data (REPORT responses)
	CalendarData      string `xml:"urn:ietf:params:xml:ns:caldav calendar-data,omitempty"`
	CalendarDataPlain string `xml:"calendar-data,omitempty"`

	// CardDAV address-data (REPORT responses)
	AddressData      string `xml:"urn:ietf:params:xml:ns:carddav address-data,omitempty"`
	AddressDataPlain string `xml:"address-data,omitempty"`

	ResourceType ResourceType `xml:"resourcetype"`

	CurrentUserPrincipal HrefWrap `xml:"current-user-principal"`

	// CalDAV calendar-home-set
	CalendarHomeSet      HrefWrap `xml:"urn:ietf:params:xml:ns:caldav calendar-home-set"`
	CalendarHomeSetPlain HrefWrap `xml:"calendar-home-set"`

	// CardDAV addressbook-home-set
	AddressbookHomeSet      HrefWrap `xml:"urn:ietf:params:xml:ns:carddav addressbook-home-set"`
	AddressbookHomeSetPlain HrefWrap `xml:"addressbook-home-set"`

	SupportedCalendarComponentSet SupportedCalendarComponentSet `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set"`
}

// GetCalendarData returns calendar-data regardless of whether the server
// included the CalDAV namespace on the element.
func (p Prop) GetCalendarData() string {
	if p.CalendarData != "" {
		return p.CalendarData
	}
	return p.CalendarDataPlain
}

// GetAddressData returns address-data regardless of namespace.
func (p Prop) GetAddressData() string {
	if p.AddressData != "" {
		return p.AddressData
	}
	return p.AddressDataPlain
}

// GetCalendarHomeSet returns the calendar-home-set href regardless of namespace.
func (p Prop) GetCalendarHomeSet() string {
	if p.CalendarHomeSet.Href != "" {
		return p.CalendarHomeSet.Href
	}
	return p.CalendarHomeSetPlain.Href
}

// GetAddressbookHomeSet returns the addressbook-home-set href regardless of namespace.
func (p Prop) GetAddressbookHomeSet() string {
	if p.AddressbookHomeSet.Href != "" {
		return p.AddressbookHomeSet.Href
	}
	return p.AddressbookHomeSetPlain.Href
}

type HrefWrap struct {
	Href string `xml:"href"`
}

// ResourceType holds the WebDAV resourcetype property.
// Elements inside <resourcetype> may come with or without a namespace prefix;
// we register both the bare name and the namespaced form so that strict
// servers (e.g. Yandex) are handled correctly.
type ResourceType struct {
	Collection  *struct{} `xml:"DAV: collection"`
	Calendar    *struct{} `xml:"urn:ietf:params:xml:ns:caldav calendar"`
	Addressbook *struct{} `xml:"urn:ietf:params:xml:ns:carddav addressbook"`

	// Fallbacks for servers that omit the namespace on these elements.
	CollectionPlain  *struct{} `xml:"collection"`
	CalendarPlain    *struct{} `xml:"calendar"`
	AddressbookPlain *struct{} `xml:"addressbook"`
}

func (r ResourceType) IsCollection() bool {
	return r.Collection != nil || r.CollectionPlain != nil
}
func (r ResourceType) IsCalendar() bool {
	return r.Calendar != nil || r.CalendarPlain != nil
}
func (r ResourceType) IsAddressbook() bool {
	return r.Addressbook != nil || r.AddressbookPlain != nil
}

// SupportedCalendarComponentSet holds the list of supported component types
// returned by the server for a calendar collection.
type SupportedCalendarComponentSet struct {
	Comps []CalComp `xml:"comp"`
}

// CalComp is a single <comp name="VEVENT"/> element.
type CalComp struct {
	Name string `xml:"name,attr"`
}

// Names returns the component names as a string slice (e.g. ["VEVENT","VTODO"]).
func (s SupportedCalendarComponentSet) Names() []string {
	out := make([]string, 0, len(s.Comps))
	for _, c := range s.Comps {
		if c.Name != "" {
			out = append(out, c.Name)
		}
	}
	return out
}
