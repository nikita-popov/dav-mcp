package tools

import (
	"context"
	"fmt"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// stub returns a ToolResult placeholder for unimplemented tools.
func stub(name string) (mcp.ToolResult, error) {
	return mcp.ToolResult{
		Content: []mcp.ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("%s: not yet implemented", name),
		}},
	}, nil
}

// session returns the active DAV session.
// If no session exists but env credentials are configured, it auto-connects.
// This handles MCP clients that run a separate discovery process (tools/list)
// and a separate process for actual tool calls — the in-memory session is
// lost between processes, so we reconnect transparently.
func session(ctx context.Context, cfg config.Config) (*dav.Session, error) {
	if s := dav.Get(); s != nil {
		mcp.Debugf("tools: reusing existing session")
		return s, nil
	}
	if cfg.DAVURL == "" {
		return nil, fmt.Errorf("not connected: call calendar_connect first, or set DAV_URL / DAV_USERNAME / DAV_PASSWORD")
	}
	mcp.Logger.Printf("tools: no session in memory — auto-connecting from env (DAV_URL=%s)", cfg.DAVURL)
	return dav.Connect(ctx, cfg.DAVURL, cfg.Username, cfg.Password)
}
