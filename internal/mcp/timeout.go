package mcp

import (
	"context"
	"errors"
	"time"
)

// DefaultToolTimeout defines maximum execution time for a tool.
// Prevents hung tools from blocking MCP server indefinitely.

var DefaultToolTimeout = 15 * time.Second

// RunWithTimeout executes a tool handler with timeout protection.

func RunWithTimeout(handler ToolHandler, args map[string]any) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultToolTimeout)
	defer cancel()

	type result struct {
		val any
		err error
	}

	ch := make(chan result, 1)

	go func() {
		v, err := handler(args)
		ch <- result{v, err}
	}()

	select {
	case r := <-ch:
		return r.val, r.err
	case <-ctx.Done():
		return nil, errors.New("tool execution timeout")
	}
}
