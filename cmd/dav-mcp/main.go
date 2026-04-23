package main

import (
	"log"

	"github.com/nikita-popov/dav-mcp/internal/config"
	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

func main() {

	cfg := config.Load()
	_ = cfg // will be used by DAV tools

	server := mcp.NewServer("dav-mcp", "0.1.0")

	server.AddTool(
		"calendar_list_events",
		"List calendar events in a given time range",
		mcp.InputSchema{
			Type: "object",
			Properties: map[string]mcp.Property{
				"start": {
					Type:        "string",
					Description: "Start of range in ISO 8601 format, e.g. 2026-04-01T00:00:00Z",
				},
				"end": {
					Type:        "string",
					Description: "End of range in ISO 8601 format, e.g. 2026-04-30T23:59:59Z",
				},
			},
			Required: []string{"start", "end"},
		},
		func(args map[string]any) (any, error) {
			// TODO: implement CalDAV REPORT request
			return mcp.ToolResult{
				Content: []mcp.ContentItem{
					{Type: "text", Text: "calendar is empty (not yet implemented)"},
				},
			}, nil
		},
	)

	server.AddTool(
		"contacts_list",
		"List all contacts from CardDAV address book",
		mcp.InputSchema{
			Type: "object",
		},
		func(args map[string]any) (any, error) {
			// TODO: implement CardDAV PROPFIND request
			return mcp.ToolResult{
				Content: []mcp.ContentItem{
					{Type: "text", Text: "contacts are empty (not yet implemented)"},
				},
			}, nil
		},
	)

	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
