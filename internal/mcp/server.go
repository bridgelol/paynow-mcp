package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

const latestProtocolVersion = "2025-11-25"

type ToolHandler func(context.Context, map[string]any) (CallToolResult, error)

type Tool struct {
	Name        string
	Title       string
	Description string
	InputSchema map[string]any
	Handler     ToolHandler
}

type CallToolResult struct {
	Content           []Content `json:"content"`
	StructuredContent any       `json:"structuredContent,omitempty"`
	IsError           bool      `json:"isError,omitempty"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Server struct {
	name    string
	version string
	tools   []Tool
	byName  map[string]Tool
}

func NewServer(name, version string, tools []Tool) *Server {
	byName := make(map[string]Tool, len(tools))
	for _, tool := range tools {
		byName[tool.Name] = tool
	}

	return &Server{
		name:    name,
		version: version,
		tools:   tools,
		byName:  byName,
	}
}

func NewToolResult(payload any) CallToolResult {
	return callToolResult(payload, false)
}

func NewToolError(payload any) CallToolResult {
	return callToolResult(payload, true)
}

func callToolResult(payload any, isError bool) CallToolResult {
	text := stringify(payload)
	result := CallToolResult{
		Content: []Content{
			{Type: "text", Text: text},
		},
		IsError: isError,
	}

	if payload != nil {
		if object, ok := payload.(map[string]any); ok {
			result.StructuredContent = object
		} else {
			result.StructuredContent = map[string]any{"result": payload}
		}
	}

	return result
}

func stringify(value any) string {
	if value == nil {
		return "null"
	}
	if text, ok := value.(string); ok {
		return text
	}

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprint(value)
	}

	return string(data)
}

type request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params,omitempty"`
}

type response struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 20*1024*1024)

	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)

	var writeMu sync.Mutex
	write := func(resp response) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return encoder.Encode(resp)
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			if writeErr := write(errorResponse(rawNull(), -32700, "Parse error", err.Error())); writeErr != nil {
				return writeErr
			}
			continue
		}

		resp, ok := s.handle(ctx, req)
		if !ok {
			continue
		}
		if err := write(resp); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Server) handle(ctx context.Context, req request) (response, bool) {
	if req.ID == nil {
		if req.Method == "notifications/initialized" || req.Method == "initialized" {
			return response{}, false
		}
		return response{}, false
	}

	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return errorResponse(req.ID, -32600, "Invalid Request", "jsonrpc must be 2.0"), true
	}

	switch req.Method {
	case "initialize":
		return successResponse(req.ID, s.initializeResult(req.Params)), true
	case "ping":
		return successResponse(req.ID, map[string]any{}), true
	case "tools/list":
		return successResponse(req.ID, map[string]any{"tools": s.toolDefinitions()}), true
	case "tools/call":
		result, err := s.callTool(ctx, req.Params)
		if err != nil {
			return errorResponse(req.ID, -32602, "Invalid params", err.Error()), true
		}
		return successResponse(req.ID, result), true
	default:
		return errorResponse(req.ID, -32601, "Method not found", req.Method), true
	}
}

func (s *Server) initializeResult(params json.RawMessage) map[string]any {
	var initParams struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &initParams)

	protocolVersion := initParams.ProtocolVersion
	if protocolVersion == "" {
		protocolVersion = latestProtocolVersion
	}

	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
		"instructions": "Use the paynow_* tools to manage a PayNow store through the PayNow Management API. Set PAYNOW_API_KEY before starting the server.",
	}
}

func (s *Server) toolDefinitions() []map[string]any {
	defs := make([]map[string]any, 0, len(s.tools))
	for _, tool := range s.tools {
		defs = append(defs, map[string]any{
			"name":        tool.Name,
			"title":       tool.Title,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	return defs
}

func (s *Server) callTool(ctx context.Context, params json.RawMessage) (CallToolResult, error) {
	var call struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return CallToolResult{}, err
	}
	if call.Name == "" {
		return CallToolResult{}, errors.New("tool name is required")
	}
	if call.Arguments == nil {
		call.Arguments = map[string]any{}
	}

	tool, ok := s.byName[call.Name]
	if !ok {
		return CallToolResult{}, fmt.Errorf("unknown tool %q", call.Name)
	}

	result, err := tool.Handler(ctx, call.Arguments)
	if err != nil {
		return NewToolError(map[string]any{"error": err.Error()}), nil
	}

	return result, nil
}

func successResponse(id *json.RawMessage, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id *json.RawMessage, code int, message string, data any) response {
	return response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func rawNull() *json.RawMessage {
	raw := json.RawMessage("null")
	return &raw
}
