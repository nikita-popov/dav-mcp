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

// store is a map of account name → Session, protected by a RW mutex.
var (
	mu    sync.RWMutex
	store = map[string]*Session{}
)

// Connect creates a DAV client, runs full discovery, stores the result
// under the given account name, and returns it.
func Connect(ctx context.Context, name, rawURL, username, password string) (*Session, error) {
	mcp.Logger.Printf("dav: connecting account=%q url=%s", name, rawURL)

	c, err := New(rawURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("dav connect: %w", err)
	}

	principal, err := DiscoverPrincipal(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("dav discover principal: %w", err)
	}
	mcp.Debugf("dav[%s]: principal=%s", name, principal)

	calHome, err := DiscoverCalendarHome(ctx, c, principal)
	if err != nil {
		return nil, fmt.Errorf("dav discover calendar home: %w", err)
	}
	mcp.Debugf("dav[%s]: calendar-home=%s", name, calHome)

	calendars, err := DiscoverCollections(ctx, c, calHome)
	if err != nil {
		return nil, fmt.Errorf("dav discover calendars: %w", err)
	}
	mcp.Debugf("dav[%s]: found %d calendars", name, len(calendars))

	caps := buildCaps(calendars)
	mcp.Logger.Printf("dav[%s]: supported components: %v", name, capsList(caps))

	// addressbook-home is optional
	abHome, err := DiscoverAddressbookHome(ctx, c, principal)
	if err != nil {
		mcp.Debugf("dav[%s]: addressbook-home not found: %v", name, err)
	} else {
		mcp.Debugf("dav[%s]: addressbook-home=%s", name, abHome)
		caps.CardDAV = true
	}

	s := &Session{
		Client:          c,
		CalendarHome:    calHome,
		AddressbookHome: abHome,
		Calendars:       calendars,
		Caps:            caps,
	}
	set(name, s)
	mcp.Logger.Printf("dav[%s]: connected — %d calendars, carddav=%v", name, len(calendars), caps.CardDAV)
	return s, nil
}

// Get returns the session for the given account name.
// An empty name returns the "default" account, or the first stored session.
// Returns nil if no session exists for the name.
func Get(name string) *Session {
	mu.RLock()
	defer mu.RUnlock()
	if name == "" {
		if s, ok := store["default"]; ok {
			return s
		}
		// return any first entry
		for _, s := range store {
			return s
		}
		return nil
	}
	return store[name]
}

// Names returns all account names that have an active session.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(store))
	for k := range store {
		out = append(out, k)
	}
	return out
}

func set(name string, s *Session) {
	mu.Lock()
	defer mu.Unlock()
	store[name] = s
}

func buildCaps(cols []Collection) Capabilities {
	comps := make(map[string]bool)
	for _, col := range cols {
		for _, n := range col.Components {
			comps[strings.ToUpper(n)] = true
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
