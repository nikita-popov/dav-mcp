package dav

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// Client is a thin authenticated WebDAV/CalDAV/CardDAV HTTP client.
type Client struct {
	BaseURL    string
	httpClient *http.Client
	username   string
	password   string
}

// New creates a Client. rawURL must include scheme and host.
func New(rawURL, username, password string) (*Client, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid DAV URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("DAV URL must use http or https, got %q", u.Scheme)
	}
	return &Client{
		BaseURL:    strings.TrimRight(rawURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		username:   username,
		password:   password,
	}, nil
}

// Do executes a DAV request with Basic Auth.
func (c *Client) Do(ctx context.Context, method, path string, headers map[string]string, body []byte) (*http.Response, []byte, error) {
	// absolute URLs pass through; relative paths are joined to BaseURL
	var target string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		target = path
	} else {
		target = c.BaseURL + "/" + strings.TrimLeft(path, "/")
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	mcp.Debugf("HTTP %s %s", method, target)
	if len(body) > 0 {
		mcp.Debugf("HTTP req body:\n%s", truncate(string(body), 2048))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%s %s: %w", method, target, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, fmt.Errorf("read response body: %w", err)
	}

	mcp.Debugf("HTTP %d %s", resp.StatusCode, target)
	if len(respBody) > 0 {
		mcp.Debugf("HTTP resp body:\n%s", truncate(string(respBody), 4096))
	}

	return resp, respBody, nil
}

// Propfind sends a PROPFIND request and decodes the multistatus response.
// depth should be "0", "1", or "infinity".
func (c *Client) Propfind(ctx context.Context, path, depth string, body []byte) (*Multistatus, error) {
	resp, data, err := c.Do(ctx, "PROPFIND", path, map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
		"Depth":        depth,
	}, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 207 {
		return nil, mapHTTPError(resp.StatusCode, path)
	}
	var ms Multistatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("propfind decode: %w", err)
	}
	return &ms, nil
}

// Report sends a REPORT request and decodes the multistatus response.
func (c *Client) Report(ctx context.Context, path string, body []byte) (*Multistatus, error) {
	resp, data, err := c.Do(ctx, "REPORT", path, map[string]string{
		"Content-Type": "application/xml; charset=utf-8",
		"Depth":        "1",
	}, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 207 {
		return nil, mapHTTPError(resp.StatusCode, path)
	}
	var ms Multistatus
	if err := xml.Unmarshal(data, &ms); err != nil {
		return nil, fmt.Errorf("report decode: %w", err)
	}
	return &ms, nil
}

// Put stores data at path with the given content-type.
// If etag is empty, If-None-Match:* is sent (safe create).
// If etag is non-empty, If-Match:<etag> is sent (safe update).
func (c *Client) Put(ctx context.Context, path, contentType, etag string, body []byte) error {
	headers := map[string]string{"Content-Type": contentType}
	if etag == "" {
		headers["If-None-Match"] = "*"
	} else {
		headers["If-Match"] = etag
	}
	resp, _, err := c.Do(ctx, "PUT", path, headers, body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapHTTPError(resp.StatusCode, path)
	}
	return nil
}

// Delete removes a resource. If etag is non-empty, If-Match:<etag> is sent.
func (c *Client) Delete(ctx context.Context, path, etag string) error {
	headers := map[string]string{}
	if etag != "" {
		headers["If-Match"] = etag
	}
	resp, _, err := c.Do(ctx, "DELETE", path, headers, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return mapHTTPError(resp.StatusCode, path)
	}
	return nil
}

// truncate caps a string for log output.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... (%d bytes truncated)", len(s)-max)
}
