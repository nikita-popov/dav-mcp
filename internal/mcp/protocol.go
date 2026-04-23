package mcp

import "encoding/json"

type Request struct {
	ID     any             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Response struct {
	ID     any `json:"id"`
	Result any `json:"result,omitempty"`
	Error  any `json:"error,omitempty"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
