package tools

import (
	"context"
	"fmt"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/dav"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

// session returns the active DAV session.
// If no session exists but env credentials are configured, it auto-connects.
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
