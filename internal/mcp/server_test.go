package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func newSrv() *Server {
	s := NewServer("test", "0.1")
	s.AddTool("echo", "echo tool",
		InputSchema{Type: "object"},
		func(ctx context.Context, args map[string]any) (any, error) {
			return ToolResult{Content: []ContentItem{{Type: "text", Text: "pong"}}}, nil
		},
	)
	return s
}

func runLines(t *testing.T, srv *Server, lines ...string) []Response {
	t.Helper()
	in := strings.NewReader(strings.Join(lines, "\n") + "\n")
	var out bytes.Buffer
	if err := srv.Run(in, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var resps []Response
	dec := json.NewDecoder(&out)
	for dec.More() {
		var r Response
		if err := dec.Decode(&r); err != nil {
			t.Fatalf("decode response: %v\noutput: %s", err, out.String())
		}
		resps = append(resps, r)
	}
	return resps
}

func TestServer_Initialize(t *testing.T) {
	resps := runLines(t, newSrv(), `{"id":1,"method":"initialize","params":{}}`)
	if len(resps) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resps))
	}
	if resps[0].Error != nil {
		t.Fatalf("unexpected error: %+v", resps[0].Error)
	}
	if resps[0].Result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestServer_ToolsList(t *testing.T) {
	resps := runLines(t, newSrv(),
		`{"id":1,"method":"initialize","params":{}}`,
		`{"id":2,"method":"tools/list","params":{}}`,
	)
	if resps[1].Error != nil {
		t.Fatalf("tools/list error: %+v", resps[1].Error)
	}
}

func TestServer_ToolsListBeforeInit(t *testing.T) {
	resps := runLines(t, newSrv(), `{"id":1,"method":"tools/list","params":{}}`)
	if resps[0].Error == nil {
		t.Fatal("expected error before initialize")
	}
	if resps[0].Error.Code != errMethodNotFound {
		t.Fatalf("expected errMethodNotFound, got %d", resps[0].Error.Code)
	}
}

func TestServer_UnknownMethod(t *testing.T) {
	resps := runLines(t, newSrv(),
		`{"id":1,"method":"initialize","params":{}}`,
		`{"id":2,"method":"nonexistent","params":{}}`,
	)
	if resps[1].Error == nil || resps[1].Error.Code != errMethodNotFound {
		t.Fatalf("expected errMethodNotFound, got %+v", resps[1].Error)
	}
}

func TestServer_ToolCall(t *testing.T) {
	resps := runLines(t, newSrv(),
		`{"id":1,"method":"initialize","params":{}}`,
		`{"id":2,"method":"tools/call","params":{"name":"echo","arguments":{}}}`,
	)
	if resps[1].Error != nil {
		t.Fatalf("unexpected error: %+v", resps[1].Error)
	}
}

func TestServer_ToolNotFound(t *testing.T) {
	resps := runLines(t, newSrv(),
		`{"id":1,"method":"initialize","params":{}}`,
		`{"id":2,"method":"tools/call","params":{"name":"ghost","arguments":{}}}`,
	)
	if resps[1].Error == nil || resps[1].Error.Code != errToolNotFound {
		t.Fatalf("expected errToolNotFound, got %+v", resps[1].Error)
	}
}

func TestServer_NotificationIgnored(t *testing.T) {
	resps := runLines(t, newSrv(),
		`{"id":1,"method":"initialize","params":{}}`,
		`{"method":"notifications/initialized"}`,
		`{"id":2,"method":"tools/list","params":{}}`,
	)
	if len(resps) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(resps))
	}
}

func TestServer_JSONRPCVersion(t *testing.T) {
	resps := runLines(t, newSrv(), `{"id":1,"method":"initialize","params":{}}`)
	if resps[0].JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc=2.0, got %q", resps[0].JSONRPC)
	}
}
