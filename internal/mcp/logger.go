package mcp

import (
	"fmt"
	"log"
	"os"
)

// Logger writes to stderr with timestamp + pid so you can correlate
// entries when multiple dav-mcp processes run side-by-side.
var Logger = log.New(os.Stderr, fmt.Sprintf("dav-mcp[%d] ", os.Getpid()), log.LstdFlags)
