package mcp

import (
	"log"
	"os"
)

// Logger used across MCP server. Output goes to stderr
// so it doesn't interfere with JSON-RPC stdout channel.

var Logger = log.New(os.Stderr, "dav-mcp ", log.LstdFlags)
