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
)

type Client struct {
	BaseURL   *url.URL
	HTTP      *http.Client
	Username  string
	Password  string
	UserAgent string
}

func New(base string, username string, password string) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return &Client{
		BaseURL:  u,
		Username: username,
		Password: password,
		UserAgent: "dav-mcp",
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) do(
	ctx context.Context,
	method string,
	path string,
	headers map[string]string,
	body io.Reader,
) (*http.Response, error) {
	u := c.BaseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if c.Username != "" {
		req.SetBasicAuth(c.Username, c.Password)
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
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

func (c *Client) Report(
	ctx context.Context,
	path string,
	body []byte,
) (*MultiStatus, error) {
	headers := map[string]string{
		"Depth":        "1",
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
	headers := map[string]string{
		"Content-Type": contentType,
	}
	if etag != "" {
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

func mapHTTPError(code int) error {
	switch code {
	case 404:
		return ErrNotFound
	case 409:
		return ErrConflict
	case 412:
		return ErrPreconditionFailed
	default:
		return fmt.Errorf("dav http error: %d", code)
	}
}

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

type Prop struct {
	DisplayName string `xml:"displayname,omitempty"`
	ETag        string `xml:"getetag,omitempty"`
}
