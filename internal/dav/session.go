package dav

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// Session holds an active DAV connection and server capabilities.
type Session struct {
	Client          *Client
	CalendarHome    string
	AddressbookHome string
	Calendars       []Collection
	// Caps is the union of all component types supported across all calendars.
	// Use Supports() to check before performing component-specific operations.
	Caps Capabilities
}

// Capabilities records which iCalendar component types the server supports.
type Capabilities struct {
	// Components maps component name ("VEVENT", "VTODO", …) to true.
	// Empty map means discovery returned no information — treat as all supported.
	Components map[string]bool
	// CardDAV indicates the server has an addressbook-home-set.
	CardDAV bool
}

// Supports returns true if the server advertises support for the given
// iCalendar component type (case-insensitive).
// If no component information was discovered, returns true (fail-open).
func (caps Capabilities) Supports(comp string) bool {
	if len(caps.Components) == 0 {
		return true
	}
	return caps.Components[strings.ToUpper(comp)]
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

	// Build capability map from the union of all calendar collections.
	caps := buildCaps(calendars)
	mcp.Logger.Printf("dav: supported components: %v", capsList(caps))

	// addressbook-home is optional
	abHome, err := DiscoverAddressbookHome(ctx, c, principal)
	if err != nil {
		mcp.Debugf("dav: addressbook-home not found: %v", err)
	} else {
		mcp.Debugf("dav: addressbook-home=%s", abHome)
		caps.CardDAV = true
	}

	s := &Session{
		Client:          c,
		CalendarHome:    calHome,
		AddressbookHome: abHome,
		Calendars:       calendars,
		Caps:            caps,
	}
	setSession(s)
	mcp.Logger.Printf("dav: connected — %d calendars, carddav=%v", len(calendars), caps.CardDAV)
	return s, nil
}

// buildCaps constructs a Capabilities from the union of all collection component sets.
func buildCaps(cols []Collection) Capabilities {
	comps := make(map[string]bool)
	for _, col := range cols {
		for _, name := range col.Components {
			comps[strings.ToUpper(name)] = true
		}
	}
	return Capabilities{Components: comps}
}

func capsList(caps Capabilities) []string {
	out := make([]string, 0, len(caps.Components))
	for k := range caps.Components {
		out = append(out, k)
	}
	return out
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
