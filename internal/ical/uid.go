package ical

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewUID generates a RFC 4122-style UID suitable for VEVENT/VTODO/VJOURNAL.
// Format: <unix-nano>-<random-hex>@dav-mcp
func NewUID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d-%s@dav-mcp", time.Now().UnixNano(), hex.EncodeToString(b))
}
