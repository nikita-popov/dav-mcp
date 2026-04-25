package tools

import (
	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// Register wires all tool groups into the MCP server.
// cfg is loaded once in main and passed down so env is read exactly once.
func Register(s *mcp.Server, cfg config.Config) {
	RegisterCalendar(s, cfg)
	RegisterContacts(s, cfg)
	RegisterTodo(s, cfg)
}
