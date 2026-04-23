package main

import (
	"log"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

func main() {

	cfg := config.Load()

	server := mcp.NewServer("dav-mcp", "0.1.0")

	tools.RegisterCalendar(server, cfg)
	tools.RegisterContacts(server, cfg)

	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
