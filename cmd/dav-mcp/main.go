package main

import (
	"log"

	"github.com/nikita-popov/dav-mcp/internal/mcp"
)

func main() {

	server := mcp.NewServer("dav-mcp", "0.1.0")

	server.AddTool(
		"calendar_list",
		"List calendar events",
		func(args map[string]any) (any, error) {

			return map[string]any{
				"content": []map[string]string{
					{
						"type": "text",
						"text": "calendar is empty",
					},
				},
			}, nil
		},
	)

	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
