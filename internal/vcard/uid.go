package vcard

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewUID generates a unique UID for a vCard.
// Format: <unix-nano>-<random-hex>@dav-mcp
func NewUID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d-%s@dav-mcp", time.Now().UnixNano(), hex.EncodeToString(b))
}
