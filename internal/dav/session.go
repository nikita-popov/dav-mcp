package dav

import (
	"context"
	"fmt"
	"sync"

	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// Session holds an active DAV connection: HTTP client + discovered home paths.
type Session struct {
	Client          *Client
	CalendarHome    string
	AddressbookHome string
	Calendars       []Collection
}

var (
	mu      sync.RWMutex
	current *Session
)

// Connect creates a DAV client, runs full discovery, stores the result as the
// active session, and returns it.
func Connect(ctx context.Context, rawURL, username, password string) (*Session, error) {
	mcp.Logger.Printf("dav: connecting to %s as %s", rawURL, username)

	c, err := New(rawURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("dav connect: %w", err)
	}

	principal, err := DiscoverPrincipal(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("dav discover principal: %w", err)
	}
	mcp.Debugf("dav: principal=%s", principal)

	calHome, err := DiscoverCalendarHome(ctx, c, principal)
	if err != nil {
		return nil, fmt.Errorf("dav discover calendar home: %w", err)
	}
	mcp.Debugf("dav: calendar-home=%s", calHome)

	calendars, err := DiscoverCollections(ctx, c, calHome)
	if err != nil {
		return nil, fmt.Errorf("dav discover calendars: %w", err)
	}
	mcp.Debugf("dav: found %d calendars", len(calendars))

	// addressbook-home is optional
	abHome, err := DiscoverAddressbookHome(ctx, c, principal)
	if err != nil {
		mcp.Debugf("dav: addressbook-home not found: %v", err)
	} else {
		mcp.Debugf("dav: addressbook-home=%s", abHome)
	}

	s := &Session{
		Client:          c,
		CalendarHome:    calHome,
		AddressbookHome: abHome,
		Calendars:       calendars,
	}
	setSession(s)
	mcp.Logger.Printf("dav: connected — %d calendars, addressbook-home=%q", len(calendars), abHome)
	return s, nil
}

// Get returns the active session, or nil if Connect has not been called yet.
func Get() *Session {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func setSession(s *Session) {
	mu.Lock()
	defer mu.Unlock()
	current = s
}
