package dav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a minimal WebDAV/CalDAV/CardDAV HTTP client.
// All methods accept a context so callers can enforce timeouts.
type Client struct {
	BaseURL   *url.URL
	HTTP      *http.Client
	Username  string
	Password  string
	UserAgent string
}

func New(base, username, password string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseURL:   u,
		Username:  username,
		Password:  password,
		UserAgent: "dav-mcp/0.1",
		HTTP:      &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// resolve builds an absolute URL from path.
// If path is already absolute (starts with http/https) it is used as-is.
func (c *Client) resolve(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.BaseURL.ResolveReference(&url.URL{Path: path}).String()
}

func (c *Client) do(
	ctx context.Context,
	method, path string,
	headers map[string]string,
	body io.Reader,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.resolve(path), body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("User-Agent", c.UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.HTTP.Do(req)
}

// Propfind sends a PROPFIND request and decodes the 207 Multi-Status response.
func (c *Client) Propfind(ctx context.Context, path, depth string, body []byte) (*MultiStatus, error) {
	resp, err := c.do(ctx, "PROPFIND", path, map[string]string{
		"Depth":        depth,
		"Content-Type": "application/xml; charset=utf-8",
		"Accept":       "application/xml",
	}, bytes.NewReader(body))
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

// Report sends a REPORT request and decodes the 207 Multi-Status response.
func (c *Client) Report(ctx context.Context, path string, body []byte) (*MultiStatus, error) {
	resp, err := c.do(ctx, "REPORT", path, map[string]string{
		"Depth":        "1",
		"Content-Type": "application/xml; charset=utf-8",
		"Accept":       "application/xml",
	}, bytes.NewReader(body))
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

// Put creates or updates a resource. Pass etag="" to skip If-Match (new resource).
func (c *Client) Put(ctx context.Context, path, contentType, etag string, body []byte) error {
	headers := map[string]string{"Content-Type": contentType}
	if etag != "" {
		headers["If-Match"] = etag
	} else {
		// Prevent overwriting an existing resource when creating new ones.
		headers["If-None-Match"] = "*"
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

// Delete removes a resource. Pass etag="" to skip If-Match.
func (c *Client) Delete(ctx context.Context, path, etag string) error {
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

// Get fetches the raw body of a resource.
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

// ---------------------------------------------------------------------------
// XML types for Multi-Status responses
// ---------------------------------------------------------------------------

// MultiStatus represents a DAV:multistatus response body.
type MultiStatus struct {
	XMLName   xml.Name        `xml:"multistatus"`
	Responses []PropResponse  `xml:"response"`
}

// PropResponse is a single DAV:response element.
type PropResponse struct {
	Href     string     `xml:"href"`
	Propstat []Propstat `xml:"propstat"`
}

// Propstat groups a set of properties with their HTTP status.
type Propstat struct {
	Status string   `xml:"status"`
	Prop   PropBody `xml:"prop"`
}

// PropBody holds all property values we care about across all request types.
// Fields are tagged with their respective XML namespaces.
type PropBody struct {
	// DAV: core
	DisplayName  string       `xml:"displayname"`
	ETag         string       `xml:"getetag"`
	ResourceType ResourceType `xml:"resourcetype"`

	// DAV: principal
	CurrentUserPrincipal HrefWrapper `xml:"current-user-principal"`

	// CalDAV
	CalendarHomeSet HrefWrapper `xml:"calendar-home-set"`
	CalendarData    string      `xml:"calendar-data"`

	// CardDAV
	AddressbookHomeSet HrefWrapper `xml:"addressbook-home-set"`
	AddressData        string      `xml:"address-data"`
}

// ResourceType holds the DAV:resourcetype property.
// The presence of the <collection/> child element is represented as a pointer —
// nil means the element is absent, non-nil means it was present.
type ResourceType struct {
	Collection *struct{} `xml:"collection"`
}

// IsCollection reports whether this resource is a DAV collection.
func (r ResourceType) IsCollection() bool { return r.Collection != nil }

// HrefWrapper wraps a single <href> child inside a property element.
type HrefWrapper struct {
	Href string `xml:"href"`
}
