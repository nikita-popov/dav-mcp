package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type ToolHandler func(args map[string]any) (any, error)

type toolEntry struct {
	Tool
	Handler ToolHandler
}

type Server struct {
	name    string
	version string
	tools   map[string]toolEntry
}

func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   map[string]toolEntry{},
	}
}

func (s *Server) AddTool(name, description string, handler ToolHandler) {

	s.tools[name] = toolEntry{
		Tool: Tool{
			Name:        name,
			Description: description,
		},
		Handler: handler,
	}
}

func (s *Server) Run() error {

	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {

		var req Request

		err := json.Unmarshal(scanner.Bytes(), &req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		switch req.Method {

		case "initialize":

			enc.Encode(Response{
				ID: req.ID,
				Result: map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]any{
						"tools": map[string]any{},
					},
					"serverInfo": map[string]any{
						"name":    s.name,
						"version": s.version,
					},
				},
			})

		case "tools/list":

			var list []Tool

			for _, t := range s.tools {
				list = append(list, t.Tool)
			}

			enc.Encode(Response{
				ID: req.ID,
				Result: map[string]any{
					"tools": list,
				},
			})

		case "tools/call":

			var p struct {
				Name string         `json:"name"`
				Args map[string]any `json:"arguments"`
			}

			json.Unmarshal(req.Params, &p)

			tool, ok := s.tools[p.Name]
			if !ok {

				enc.Encode(Response{
					ID:    req.ID,
					Error: "tool not found",
				})

				continue
			}

			res, err := tool.Handler(p.Args)

			if err != nil {

				enc.Encode(Response{
					ID:    req.ID,
					Error: err.Error(),
				})

				continue
			}

			enc.Encode(Response{
				ID:     req.ID,
				Result: res,
			})
		}
	}

	return scanner.Err()
}
