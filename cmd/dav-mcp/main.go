package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

// version is set at build time via -ldflags "-X main.version=v1.2.3".
// Falls back to "dev" for local builds.
var version = "dev"

func main() {
	log := mcp.Logger

	// ── startup banner ────────────────────────────────────────────────────────
	log.Printf("START pid=%d version=%s", os.Getpid(), version)

	cfg := config.Load()
	if cfg.DAVURL != "" {
		log.Printf("ENV DAV_URL=%s DAV_USERNAME=%s DAV_PASSWORD=<set=%v>",
			cfg.DAVURL, cfg.Username, cfg.Password != "")
	} else {
		log.Printf("ENV DAV_URL=(not set) — will require calendar_connect args")
	}

	// ── exit banner via defer + signal ────────────────────────────────────────
	defer log.Printf("EXIT pid=%d", os.Getpid())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		s := <-sig
		log.Printf("SIGNAL %s — shutting down", s)
		os.Exit(0)
	}()

	// ── server ────────────────────────────────────────────────────────────────
	server := mcp.NewServer("dav-mcp", version)
	tools.Register(server, cfg)

	log.Printf("READY — waiting for requests on stdin")

	if err := server.RunStdio(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}
