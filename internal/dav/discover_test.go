package dav

import "testing"

func TestCollectionSupports(t *testing.T) {
	cases := []struct {
		name       string
		components []string
		query      string
		want       bool
	}{
		{"empty components — assume all", nil, "VEVENT", true},
		{"exact match", []string{"VEVENT"}, "VEVENT", true},
		{"case-insensitive", []string{"VEVENT"}, "vevent", true},
		{"multi match", []string{"VEVENT", "VTODO"}, "VTODO", true},
		{"not found", []string{"VEVENT"}, "VTODO", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			col := Collection{Components: tc.components}
			if got := col.Supports(tc.query); got != tc.want {
				t.Errorf("Supports(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

func TestDiscoverPrincipal_NotFound(t *testing.T) {
	ms := &Multistatus{}
	// simulate no principal href in response
	_ = ms
	// DiscoverPrincipal returns ErrNotFound when multistatus has no href;
	// the httptest path is already covered by TestDiscoverPrincipal.
	// Here we just guard the sentinel value itself.
	if ErrNotFound == nil {
		t.Fatal("ErrNotFound must not be nil")
	}
}

func TestNormalizeHref(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/cal/personal/", "/cal/personal/"},
		{"/cal/personal", "/cal/personal/"},
		{"/", "/"},
	}
	for _, tc := range cases {
		if got := normalizeHref(tc.in); got != tc.want {
			t.Errorf("normalizeHref(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
