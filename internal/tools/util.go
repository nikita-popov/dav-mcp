package tools

import (
	"context"
	"fmt"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// session returns the active DAV session, auto-connecting from env if needed.
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

// requireComponent checks that the server supports a given iCalendar component
// type. Returns a ToolResult with an explanatory message if not supported, so
// the handler can return early without attempting the operation.
//
// Usage:
//
//	if result, ok := requireComponent(sess, "VTODO"); !ok {
//	    return result, nil
//	}
func requireComponent(sess *dav.Session, comp string) (mcp.ToolResult, bool) {
	if sess.Caps.Supports(comp) {
		return mcp.ToolResult{}, true
	}
	return mcp.ToolResult{
		Content: []mcp.ContentItem{{
			Type: "text",
			Text: fmt.Sprintf("%s is not supported by this CalDAV server.", comp),
		}},
	}, false
}

// requireCardDAV checks that the server supports CardDAV (has addressbook-home).
func requireCardDAV(sess *dav.Session) (mcp.ToolResult, bool) {
	if sess.Caps.CardDAV {
		return mcp.ToolResult{}, true
	}
	return mcp.ToolResult{
		Content: []mcp.ContentItem{{
			Type: "text",
			Text: "CardDAV (contacts) is not supported by this server.",
		}},
	}, false
}
