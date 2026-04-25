package dav

import (
	"sort"
	"testing"
)

// --- Capabilities.Supports ---

func TestSupports_KnownComponent(t *testing.T) {
	caps := Capabilities{Components: map[string]bool{"VEVENT": true, "VTODO": true}}
	if !caps.Supports("VEVENT") {
		t.Error("expected VEVENT to be supported")
	}
	if !caps.Supports("vevent") {
		t.Error("expected case-insensitive match")
	}
	if caps.Supports("VJOURNAL") {
		t.Error("VJOURNAL should not be supported")
	}
}

func TestSupports_EmptyComponents_FailOpen(t *testing.T) {
	caps := Capabilities{Components: map[string]bool{}}
	if !caps.Supports("VEVENT") {
		t.Error("empty components map should fail-open")
	}
	if !caps.Supports("ANYTHING") {
		t.Error("empty components map should fail-open for any value")
	}
}

func TestSupports_NilComponents_FailOpen(t *testing.T) {
	caps := Capabilities{}
	if !caps.Supports("VEVENT") {
		t.Error("nil components map should fail-open")
	}
}

// --- buildCaps ---

func TestBuildCaps_MergesAllCollections(t *testing.T) {
	cols := []Collection{
		{Href: "/cal/personal/", Components: []string{"VEVENT"}},
		{Href: "/cal/tasks/", Components: []string{"VTODO"}},
		{Href: "/cal/journal/", Components: []string{"VJOURNAL", "VEVENT"}},
	}
	caps := buildCaps(cols)
	for _, want := range []string{"VEVENT", "VTODO", "VJOURNAL"} {
		if !caps.Components[want] {
			t.Errorf("expected %s in caps", want)
		}
	}
}

func TestBuildCaps_Empty(t *testing.T) {
	caps := buildCaps(nil)
	if len(caps.Components) != 0 {
		t.Errorf("expected empty components, got %v", caps.Components)
	}
	// fail-open: empty map → Supports returns true
	if !caps.Supports("VEVENT") {
		t.Error("empty caps should fail-open")
	}
}

// --- capsList ---

func TestCapsList_ReturnsAllKeys(t *testing.T) {
	caps := Capabilities{Components: map[string]bool{"VEVENT": true, "VTODO": true}}
	list := capsList(caps)
	sort.Strings(list)
	if len(list) != 2 || list[0] != "VEVENT" || list[1] != "VTODO" {
		t.Errorf("unexpected capsList: %v", list)
	}
}

func TestCapsList_Empty(t *testing.T) {
	list := capsList(Capabilities{})
	if len(list) != 0 {
		t.Errorf("expected empty list, got %v", list)
	}
}

// --- Names / set / Get ---

func TestNamesAndGet(t *testing.T) {
	set("test-alice", &Session{CalendarHome: "/cal/alice/"})
	set("test-bob", &Session{CalendarHome: "/cal/bob/"})

	names := Names()
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["test-alice"] {
		t.Error("expected test-alice in Names()")
	}
	if !found["test-bob"] {
		t.Error("expected test-bob in Names()")
	}

	s := Get("test-alice")
	if s == nil {
		t.Fatal("Get(test-alice) returned nil")
	}
	if s.CalendarHome != "/cal/alice/" {
		t.Errorf("CalendarHome=%q", s.CalendarHome)
	}

	if Get("test-nonexistent") != nil {
		t.Error("expected nil for unknown account")
	}
}
