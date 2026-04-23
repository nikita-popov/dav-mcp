package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

const jsonrpcVersion = "2.0"

// Error codes per JSON-RPC 2.0 / MCP spec
const (
	errParseError     = -32700
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errToolNotFound   = -32001
)

type ToolHandler func(args map[string]any) (any, error)

type toolEntry struct {
	Tool
	Handler ToolHandler
}

type Server struct {
	name        string
	version     string
	tools       map[string]toolEntry
	initialized bool
}

func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		tools:   map[string]toolEntry{},
	}
}

func (s *Server) AddTool(name, description string, schema InputSchema, handler ToolHandler) {
	s.tools[name] = toolEntry{
		Tool: Tool{
			Name:        name,
			Description: description,
			InputSchema: schema,
		},
		Handler: handler,
	}
}

func (s *Server) respond(enc *json.Encoder, id any, result any) {
	enc.Encode(Response{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Result:  result,
	})
}

func (s *Server) respondErr(enc *json.Encoder, id any, code int, msg string) {
	enc.Encode(Response{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	})
}

func (s *Server) Run() error {

	reader := bufio.NewReaderSize(os.Stdin, 1<<20) // 1 MB — enough for large vCard/ics
	enc := json.NewEncoder(os.Stdout)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if len(line) == 0 {
				break
			}
		}
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintln(os.Stderr, "parse error:", err)
			s.respondErr(enc, nil, errParseError, err.Error())
			continue
		}

		// Notifications have no id — handle and skip response
		if req.ID == nil {
			// e.g. notifications/initialized — silently acknowledge
			continue
		}

		switch req.Method {

		case "initialize":
			s.initialized = true
			s.respond(enc, req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
				"serverInfo": map[string]any{
					"name":    s.name,
					"version": s.version,
				},
			})

		case "tools/list":
			if !s.initialized {
				s.respondErr(enc, req.ID, errMethodNotFound, "server not initialized")
				continue
			}
			list := make([]Tool, 0, len(s.tools))
			for _, t := range s.tools {
				list = append(list, t.Tool)
			}
			s.respond(enc, req.ID, map[string]any{"tools": list})

		case "tools/call":
			if !s.initialized {
				s.respondErr(enc, req.ID, errMethodNotFound, "server not initialized")
				continue
			}

			var p struct {
				Name string         `json:"name"`
				Args map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				s.respondErr(enc, req.ID, errInvalidParams, err.Error())
				continue
			}

			tool, ok := s.tools[p.Name]
			if !ok {
				s.respondErr(enc, req.ID, errToolNotFound, "tool not found: "+p.Name)
				continue
			}

			res, err := tool.Handler(p.Args)
			if err != nil {
				// Tool errors are returned as result with isError:true per MCP spec
				s.respond(enc, req.ID, ToolResult{
					Content: []ContentItem{{Type: "text", Text: err.Error()}},
					IsError: true,
				})
				continue
			}
			s.respond(enc, req.ID, res)

		default:
			s.respondErr(enc, req.ID, errMethodNotFound, "method not found: "+req.Method)
		}
	}

	return nil
}
