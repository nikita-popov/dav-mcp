package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunWithTimeout_Success(t *testing.T) {
	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return "ok", nil
	}
	v, err := RunWithTimeout(handler, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v != "ok" {
		t.Fatalf("expected 'ok', got %v", v)
	}
}

func TestRunWithTimeout_HandlerError(t *testing.T) {
	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return nil, errors.New("dav: not found")
	}
	_, err := RunWithTimeout(handler, nil)
	if err == nil || err.Error() != "dav: not found" {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestRunWithTimeout_Timeout(t *testing.T) {
	old := DefaultToolTimeout
	DefaultToolTimeout = 20 * time.Millisecond
	defer func() { DefaultToolTimeout = old }()

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			return "late", nil
		}
	}
	_, err := RunWithTimeout(handler, nil)
	if err == nil || err.Error() != "tool execution timeout" {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestRunWithTimeout_CtxPassedToHandler(t *testing.T) {
	var got context.Context
	handler := func(ctx context.Context, args map[string]any) (any, error) {
		got = ctx
		return nil, nil
	}
	RunWithTimeout(handler, nil)
	if got == nil {
		t.Fatal("context was not passed to handler")
	}
	if got.Done() == nil {
		t.Fatal("context has no deadline/cancel channel")
	}
}
