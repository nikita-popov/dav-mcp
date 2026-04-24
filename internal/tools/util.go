package tools

import (
	"context"
	"fmt"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// session returns the DAV session for the given account name.
// If the session is not yet connected, it auto-connects using cfg.
// An empty accountName selects the primary/default account.
func session(ctx context.Context, cfg config.Config, accountName string) (*dav.Session, error) {
	if s := dav.Get(accountName); s != nil {
		mcp.Debugf("tools: reusing session account=%q", accountName)
		return s, nil
	}

	acc, err := cfg.Account(accountName)
	if err != nil {
		return nil, err
	}
	if acc.URL == "" {
		return nil, fmt.Errorf(
			"not connected: no credentials for account %q. "+
				"Call calendar_connect, or set DAV_URL / DAV_ACCOUNTS",
			accountName,
		)
	}

	mcp.Logger.Printf("tools: auto-connecting account=%q url=%s", acc.Name, acc.URL)
	return dav.Connect(ctx, acc.Name, acc.URL, acc.Username, acc.Password)
}

// requireComponent checks that the server supports a given iCalendar component
// type. Returns a ToolResult with an explanatory message if not supported.
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

// requireCardDAV checks that the server supports CardDAV.
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
