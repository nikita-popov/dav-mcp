package dav

import (
	"context"
	"fmt"
	"sync"
)

// Session holds an active DAV connection: HTTP client + discovered home paths.
// Package-level singleton; tools call Connect or Reconnect to populate it.
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

// Connect creates a DAV client, runs the full discovery sequence
// (principal → calendar-home → calendar list), stores the result as the
// active session, and returns it.
func Connect(ctx context.Context, rawURL, username, password string) (*Session, error) {
	c, err := New(rawURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("dav connect: %w", err)
	}

	principal, err := DiscoverPrincipal(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("dav discover principal: %w", err)
	}

	calHome, err := DiscoverCalendarHome(ctx, c, principal)
	if err != nil {
		return nil, fmt.Errorf("dav discover calendar home: %w", err)
	}

	calendars, err := DiscoverCollections(ctx, c, calHome)
	if err != nil {
		return nil, fmt.Errorf("dav discover calendars: %w", err)
	}

	// addressbook-home is optional — CardDAV servers may not expose it
	abHome, _ := DiscoverAddressbookHome(ctx, c, principal)

	s := &Session{
		Client:          c,
		CalendarHome:    calHome,
		AddressbookHome: abHome,
		Calendars:       calendars,
	}
	setSession(s)
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
