package dav

import (
	"bytes"
	"context"
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
	BaseURL  string
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
		BaseURL:  strings.TrimRight(rawURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
		username:   username,
		password:   password,
	}, nil
}

// Do executes a DAV request with Basic Auth. The body (if non-nil) is read once.
func (c *Client) Do(ctx context.Context, method, path string, headers map[string]string, body []byte) (*http.Response, []byte, error) {
	target := c.BaseURL + path

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

// Resolve returns an absolute URL by joining path onto BaseURL.
func (c *Client) Resolve(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.BaseURL + "/" + strings.TrimLeft(path, "/")
}

// truncate caps a string for log output.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... (%d bytes truncated)", len(s)-max)
}
