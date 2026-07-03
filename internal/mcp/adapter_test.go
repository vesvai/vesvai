package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMCPTool_Name(t *testing.T) {
	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tool := NewMCPTool(client, Tool{
		Name:        "my_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"integer"}}}`),
	})

	if tool.Name() != "my_tool" {
		t.Errorf("expected name 'my_tool', got %q", tool.Name())
	}
}

func TestMCPTool_Description(t *testing.T) {
	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tool := NewMCPTool(client, Tool{
		Name:        "my_tool",
		Description: "A test tool",
	})

	if tool.Description() != "A test tool" {
		t.Errorf("expected description 'A test tool', got %q", tool.Description())
	}
}

func TestMCPTool_Schema(t *testing.T) {
	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	schemaJSON := `{"type":"object","properties":{"x":{"type":"integer"}}}`
	tool := NewMCPTool(client, Tool{
		Name:        "my_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(schemaJSON),
	})

	schema := tool.Schema()
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	if schema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", schema["type"])
	}
}

func TestMCPTool_Schema_Nil(t *testing.T) {
	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tool := NewMCPTool(client, Tool{
		Name: "my_tool",
	})

	schema := tool.Schema()
	if schema == nil {
		t.Fatal("expected non-nil schema (should have default)")
	}

	if schema["type"] != "object" {
		t.Errorf("expected default type 'object', got %v", schema["type"])
	}
}

func TestMCPTool_Handle(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodToolsCall, func(params any) (any, *RPCError) {
		callParams := params.(map[string]any)
		name := callParams["name"].(string)
		args := callParams["arguments"].(map[string]any)

		if name == "greet" {
			name, _ := args["name"].(string)
			return CallToolResult{
				Content: []ToolContent{
					{Type: "text", Text: "Hello, " + name + "!"},
				},
			}, nil
		}

		return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: "unknown tool"}
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tool := NewMCPTool(client, Tool{
		Name:        "greet",
		Description: "Greet someone",
	})

	result, err := tool.Handle(context.Background(), map[string]any{
		"name": "World",
	})
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", result)
	}
}

func TestMCPTool_Handle_Error(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodToolsCall, func(params any) (any, *RPCError) {
		return CallToolResult{
			Content: []ToolContent{
				{Type: "text", Text: "invalid input"},
			},
			IsError: true,
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tool := NewMCPTool(client, Tool{
		Name: "failing",
	})

	_, err := tool.Handle(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error from failing tool")
	}

	expected := `MCP tool "failing" failed: tool error: invalid input`
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestMCPTool_Handle_MultipleContent(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodToolsCall, func(params any) (any, *RPCError) {
		return CallToolResult{
			Content: []ToolContent{
				{Type: "text", Text: "line 1"},
				{Type: "text", Text: "line 2"},
			},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tool := NewMCPTool(client, Tool{
		Name: "multi",
	})

	result, err := tool.Handle(context.Background(), nil)
	if err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	expected := "line 1\nline 2"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDiscoverTools(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodToolsList, func(params any) (any, *RPCError) {
		return ListToolsResult{
			Tools: []Tool{
				{Name: "tool_a", Description: "Tool A"},
				{Name: "tool_b", Description: "Tool B"},
				{Name: "tool_c", Description: "Tool C"},
			},
		}, nil
	})

	tools, err := DiscoverTools(context.Background(), mock)
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}

	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name()
	}

	expected := []string{"tool_a", "tool_b", "tool_c"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected tool name %q at index %d, got %q", expected[i], i, name)
		}
	}
}

func TestDiscoverTools_Empty(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodToolsList, func(params any) (any, *RPCError) {
		return ListToolsResult{Tools: []Tool{}}, nil
	})

	tools, err := DiscoverTools(context.Background(), mock)
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}

	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}

func TestMCPServerConfig(t *testing.T) {
	config := MCPServerConfig{
		Command: "node",
		Args:    []string{"server.js"},
		Env:     []string{"NODE_ENV=production"},
	}

	if config.Command != "node" {
		t.Errorf("expected command 'node', got %q", config.Command)
	}
	if len(config.Args) != 1 || config.Args[0] != "server.js" {
		t.Errorf("expected args [server.js], got %v", config.Args)
	}
	if len(config.Env) != 1 || config.Env[0] != "NODE_ENV=production" {
		t.Errorf("expected env [NODE_ENV=production], got %v", config.Env)
	}
}
