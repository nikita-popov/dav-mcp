package mcp

import (
	"context"
	"errors"
	"time"
)

// DefaultToolTimeout defines maximum execution time for a tool.
// Prevents hung tools (e.g. stalled DAV requests) from blocking the server.
var DefaultToolTimeout = 15 * time.Second

// RunWithTimeout executes a tool handler with a deadline.
// The context is passed to the handler so HTTP clients can respect cancellation.
func RunWithTimeout(handler ToolHandler, args map[string]any) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultToolTimeout)
	defer cancel()

	type result struct {
		val any
		err error
	}

	ch := make(chan result, 1)

	go func() {
		v, err := handler(ctx, args)
		ch <- result{v, err}
	}()

	select {
	case r := <-ch:
		return r.val, r.err
	case <-ctx.Done():
		return nil, errors.New("tool execution timeout")
	}
}
