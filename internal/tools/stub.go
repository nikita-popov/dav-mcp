package tools

import "github.com/nikita-popov/dav-mcp/internal/mcp"

// stub returns a not-yet-implemented ToolResult.
// Remove call sites as real implementations are added.
func stub(tool string) mcp.ToolResult {
	return mcp.ToolResult{
		Content: []mcp.ContentItem{
			{Type: "text", Text: tool + ": not yet implemented"},
		},
	}
}
