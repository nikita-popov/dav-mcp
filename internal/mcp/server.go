package mcp

import (
	"encoding/json"
	"io"
	"os"
	"sort"
	"time"
)

const jsonrpcVersion = "2.0"

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
		Tool:    Tool{Name: name, Description: description, InputSchema: schema},
		Handler: handler,
	}
}

// Run reads JSON-RPC requests from r and writes responses to w.
// Returns when r reaches EOF or a read error occurs.
func (s *Server) Run(r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)

	for {
		var req Request
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				Logger.Println("EOF — client closed connection")
				return nil
			}
			Logger.Println("parse error:", err)
			s.respondErr(enc, nil, errParseError, err.Error())
			continue
		}

		// Notifications (no id) — log and skip per MCP spec.
		if req.ID == nil {
			Logger.Printf("NOTIFY method=%s (ignored)", req.Method)
			continue
		}

		Logger.Printf("REQ id=%v method=%s", req.ID, req.Method)

		switch req.Method {
		case "initialize":
			s.initialized = true
			Logger.Printf("initialize: marking server ready")
			s.respond(enc, req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": s.name, "version": s.version},
			})

		case "tools/list":
			if !s.initialized {
				Logger.Printf("tools/list: rejected — not initialized")
				s.respondErr(enc, req.ID, errMethodNotFound, "server not initialized")
				continue
			}
			tools := s.listTools()
			Logger.Printf("tools/list: returning %d tools", len(tools))
			s.respond(enc, req.ID, map[string]any{"tools": tools})

		case "tools/call":
			if !s.initialized {
				Logger.Printf("tools/call: rejected — not initialized")
				s.respondErr(enc, req.ID, errMethodNotFound, "server not initialized")
				continue
			}
			var p struct {
				Name string         `json:"name"`
				Args map[string]any `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &p); err != nil {
				Logger.Printf("tools/call: bad params: %v", err)
				s.respondErr(enc, req.ID, errInvalidParams, err.Error())
				continue
			}
			tool, ok := s.tools[p.Name]
			if !ok {
				Logger.Printf("tools/call: unknown tool %q", p.Name)
				s.respondErr(enc, req.ID, errToolNotFound, "tool not found: "+p.Name)
				continue
			}
			start := time.Now()
			Logger.Printf("tool start: %s args=%v", p.Name, sanitizeArgs(p.Args))
			res, err := RunWithTimeout(tool.Handler, p.Args)
			dur := time.Since(start)
			if err != nil {
				Logger.Printf("tool error: %s duration=%s err=%v", p.Name, dur, err)
				s.respond(enc, req.ID, ToolResult{
					Content: []ContentItem{{Type: "text", Text: err.Error()}},
					IsError: true,
				})
				continue
			}
			Logger.Printf("tool ok: %s duration=%s", p.Name, dur)
			s.respond(enc, req.ID, res)

		default:
			Logger.Printf("unknown method: %s", req.Method)
			s.respondErr(enc, req.ID, errMethodNotFound, "method not found: "+req.Method)
		}
	}
}

// RunStdio is the entry point for production use (stdin/stdout).
func (s *Server) RunStdio() error {
	return s.Run(os.Stdin, os.Stdout)
}

func (s *Server) respond(enc *json.Encoder, id any, result any) {
	enc.Encode(Response{JSONRPC: jsonrpcVersion, ID: id, Result: result})
}

func (s *Server) respondErr(enc *json.Encoder, id any, code int, msg string) {
	enc.Encode(Response{JSONRPC: jsonrpcVersion, ID: id, Error: &RPCError{Code: code, Message: msg}})
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

// sanitizeArgs returns args map with password/token fields redacted.
func sanitizeArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return args
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		switch k {
		case "password", "token", "secret", "api_key":
			out[k] = "<redacted>"
		default:
			out[k] = v
		}
	}
	return out
}
