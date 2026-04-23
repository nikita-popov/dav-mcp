package tools

import (
	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// Register wires all tool groups into the MCP server.
func Register(s *mcp.Server) {
	cfg := config.Load()
	RegisterCalendar(s, cfg)
	RegisterContacts(s, cfg)
}
