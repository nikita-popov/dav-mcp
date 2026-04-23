package mcp

import (
	"encoding/json"
	"os"
	"sort"
	"time"
)

const jsonrpcVersion = "2.0"

// Error codes per JSON-RPC 2.0 / MCP spec
const (
	errParseError     = -32700
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errToolNotFound   = -32001
)

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
	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for {
		var req Request

		if err := dec.Decode(&req); err != nil {
			Logger.Println("parse error:", err)
			s.respondErr(enc, nil, errParseError, err.Error())
			continue
		}

		// Notifications have no id — silently ignore
		if req.ID == nil {
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
			s.respond(enc, req.ID, map[string]any{
				"tools": s.listTools(),
			})

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

			start := time.Now()
			Logger.Println("tool start:", p.Name)
			res, err := RunWithTimeout(tool.Handler, p.Args)
			Logger.Println("tool end:", p.Name, "duration=", time.Since(start))

			if err != nil {
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
}

func (s *Server) listTools() []Tool {
	keys := make([]string, 0, len(s.tools))
	for k := range s.tools {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	list := make([]Tool, 0, len(keys))
	for _, k := range keys {
		list = append(list, s.tools[k].Tool)
	}
	return list
}
