package mcp

import (
	"fmt"
	"io"
	"log"
	"os"
)

var (
	// Logger is always active — startup banners, REQ, tool start/end, errors.
	Logger = log.New(os.Stderr, fmt.Sprintf("dav-mcp[%d] ", os.Getpid()), log.LstdFlags)

	// Debug is active only when DAV_DEBUG=1. Use for HTTP bodies, XML, raw iCal/vCard.
	Debug *log.Logger
)

func init() {
	if os.Getenv("DAV_DEBUG") == "1" {
		Debug = log.New(os.Stderr, fmt.Sprintf("dav-mcp[%d] DBG ", os.Getpid()), log.LstdFlags|log.Lshortfile)
	} else {
		Debug = log.New(io.Discard, "", 0)
	}
}

// Debugf logs only when DAV_DEBUG=1.
func Debugf(format string, args ...any) {
	Debug.Printf(format, args...)
}
