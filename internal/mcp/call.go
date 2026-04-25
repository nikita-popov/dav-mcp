package mcp

import (
	"context"
	"fmt"
)

// CallTool invokes a registered tool handler directly, bypassing JSON-RPC
// framing. Intended for use in tests.
func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) (any, error) {
	tool, ok := s.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Handler(ctx, args)
}
