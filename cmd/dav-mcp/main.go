package main

import (
	"log"

	"github.com/nikita-popov/dav-mcp/internal/mcp"
	"github.com/nikita-popov/dav-mcp/internal/tools"
)

func main() {
	server := mcp.NewServer("dav-mcp", "0.1.0")
	tools.Register(server)

	if err := server.RunStdio(); err != nil {
		log.Fatal(err)
	}
}
