package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestServerInitializeAndToolsList(t *testing.T) {
	server := NewServer("test-server", "0.0.1", []Tool{
		{
			Name:        "test_tool",
			Title:       "Test tool",
			Description: "A test tool.",
			InputSchema: map[string]any{"type": "object"},
			Handler: func(context.Context, map[string]any) (CallToolResult, error) {
				return NewToolResult(map[string]any{"ok": true}), nil
			},
		},
	})

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"test_tool","arguments":{}}}`,
		"",
	}, "\n")

	var output bytes.Buffer
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d responses: %s", len(lines), output.String())
	}

	var initResp map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("decode initialize response: %v", err)
	}
	result := initResp["result"].(map[string]any)
	if result["protocolVersion"] != "2025-06-18" {
		t.Fatalf("protocolVersion = %#v", result["protocolVersion"])
	}

	var listResp map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("decode tools/list response: %v", err)
	}
	listResult := listResp["result"].(map[string]any)
	tools := listResult["tools"].([]any)
	if len(tools) != 1 {
		t.Fatalf("tool count = %d", len(tools))
	}

	var callResp map[string]any
	if err := json.Unmarshal([]byte(lines[2]), &callResp); err != nil {
		t.Fatalf("decode tools/call response: %v", err)
	}
	callResult := callResp["result"].(map[string]any)
	if callResult["isError"] == true {
		t.Fatalf("tool call returned error: %#v", callResult)
	}
}
