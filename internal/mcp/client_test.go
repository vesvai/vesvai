package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

type mockTransport struct {
	mu sync.Mutex

	handlers map[string]func(params any) (any, *RPCError)

	notifications []JSONRPCNotification

	started bool
	closed  bool

	notifHandler func(JSONRPCNotification)
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		handlers: make(map[string]func(params any) (any, *RPCError)),
	}
}

func (m *mockTransport) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	return nil
}

func (m *mockTransport) SendRequest(_ context.Context, method string, params any) (*JSONRPCResponse, error) {
	m.mu.Lock()
	handler, ok := m.handlers[method]
	m.mu.Unlock()

	if !ok {
		return &JSONRPCResponse{
			JSONRPC: jsonrpcVersion,
			Error: &RPCError{
				Code:    ErrCodeMethodNotFound,
				Message: fmt.Sprintf("method not found: %s", method),
			},
		}, nil
	}

	var normalized any
	if params != nil {
		data, err := json.Marshal(params)
		if err == nil {
			json.Unmarshal(data, &normalized)
		} else {
			normalized = params
		}
	}

	result, rpcErr := handler(normalized)
	return &JSONRPCResponse{
		JSONRPC: jsonrpcVersion,
		Result:  result,
		Error:   rpcErr,
	}, nil
}

func (m *mockTransport) SendNotification(_ context.Context, _ string, _ any) error {
	return nil
}

func (m *mockTransport) SetNotificationHandler(handler func(notification JSONRPCNotification)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifHandler = handler
}

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockTransport) setHandler(method string, handler func(params any) (any, *RPCError)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[method] = handler
}

func TestClient_Connect(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{ListChanged: true},
			},
			ServerInfo: Implementation{
				Name:    "test-server",
				Version: "1.0.0",
			},
			Instructions: "Test server",
		}, nil
	})

	client := NewClient(mock)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !mock.started {
		t.Fatal("transport was not started")
	}

	info := client.ServerInfo()
	if info.Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %q", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected server version '1.0.0', got %q", info.Version)
	}

	caps := client.Capabilities()
	if caps.Tools == nil || !caps.Tools.ListChanged {
		t.Error("expected tools capability with listChanged")
	}

	if client.Instructions() != "Test server" {
		t.Errorf("expected instructions 'Test server', got %q", client.Instructions())
	}
}

func TestClient_ListTools(t *testing.T) {
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
				{
					Name:        "get_weather",
					Description: "Get weather for a location",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
				},
				{
					Name:        "search",
					Description: "Search the web",
					InputSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
				},
			},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	if tools[0].Name != "get_weather" {
		t.Errorf("expected first tool name 'get_weather', got %q", tools[0].Name)
	}
	if tools[1].Name != "search" {
		t.Errorf("expected second tool name 'search', got %q", tools[1].Name)
	}
}

func TestClient_CallTool(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodToolsCall, func(params any) (any, *RPCError) {
		callParams, ok := params.(map[string]any)
		if !ok {
			return nil, &RPCError{Code: ErrCodeInvalidParams, Message: "invalid params"}
		}

		name, _ := callParams["name"].(string)
		args, _ := callParams["arguments"].(map[string]any)

		if name == "echo" {
			msg, _ := args["message"].(string)
			return CallToolResult{
				Content: []ToolContent{
					{Type: "text", Text: fmt.Sprintf("echo: %s", msg)},
				},
			}, nil
		}

		return nil, &RPCError{Code: ErrCodeMethodNotFound, Message: fmt.Sprintf("unknown tool: %s", name)}
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	result, err := client.CallTool(context.Background(), "echo", map[string]any{
		"message": "hello",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Fatal("expected no error in result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "echo: hello" {
		t.Errorf("expected 'echo: hello', got %q", result.Content[0].Text)
	}
}

func TestClient_CallTool_Error(t *testing.T) {
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
				{Type: "text", Text: "something went wrong"},
			},
			IsError: true,
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	_, err := client.CallTool(context.Background(), "failing_tool", nil)
	if err == nil {
		t.Fatal("expected error from failing tool")
	}

	expected := "tool error: something went wrong"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestClient_ListResources(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodResourcesList, func(params any) (any, *RPCError) {
		return ListResourcesResult{
			Resources: []Resource{
				{
					URI:      "file:///README.md",
					Name:     "README",
					MimeType: "text/markdown",
				},
			},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	resources, err := client.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources failed: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}

	if resources[0].URI != "file:///README.md" {
		t.Errorf("expected URI 'file:///README.md', got %q", resources[0].URI)
	}
}

func TestClient_ReadResource(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodResourcesRead, func(params any) (any, *RPCError) {
		return ReadResourceResult{
			Contents: []ResourceContents{
				{
					URI:      "file:///README.md",
					MimeType: "text/markdown",
					Text:     "# Hello World",
				},
			},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	result, err := client.ReadResource(context.Background(), "file:///README.md")
	if err != nil {
		t.Fatalf("ReadResource failed: %v", err)
	}

	if len(result.Contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(result.Contents))
	}

	if result.Contents[0].Text != "# Hello World" {
		t.Errorf("expected '# Hello World', got %q", result.Contents[0].Text)
	}
}

func TestClient_ListPrompts(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodPromptsList, func(params any) (any, *RPCError) {
		return ListPromptsResult{
			Prompts: []Prompt{
				{
					Name:        "code_review",
					Description: "Review code",
					Arguments: []PromptArgument{
						{Name: "code", Description: "Code to review", Required: true},
					},
				},
			},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	prompts, err := client.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts failed: %v", err)
	}

	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(prompts))
	}

	if prompts[0].Name != "code_review" {
		t.Errorf("expected prompt name 'code_review', got %q", prompts[0].Name)
	}
}

func TestClient_GetPrompt(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(params any) (any, *RPCError) {
		return InitializeResult{
			ProtocolVersion: ProtocolVersion,
			ServerInfo:      Implementation{Name: "test", Version: "1.0.0"},
		}, nil
	})

	mock.setHandler(MethodPromptsGet, func(params any) (any, *RPCError) {
		return GetPromptResult{
			Description: "Code review prompt",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: PromptContent{
						Type: "text",
						Text: "Review this code: def hello(): pass",
					},
				},
			},
		}, nil
	})

	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	result, err := client.GetPrompt(context.Background(), "code_review", map[string]string{
		"code": "def hello(): pass",
	})
	if err != nil {
		t.Fatalf("GetPrompt failed: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result.Messages))
	}

	if result.Messages[0].Content.Text != "Review this code: def hello(): pass" {
		t.Errorf("unexpected prompt text: %q", result.Messages[0].Content.Text)
	}
}

func TestClient_Close(t *testing.T) {
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

	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !mock.closed {
		t.Error("transport was not closed")
	}
}

func TestRequestID_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		id       RequestID
		expected string
	}{
		{"int", IntID(42), "42"},
		{"string", StringID("abc-123"), `"abc-123"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.id)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestRequestID_UnmarshalJSON(t *testing.T) {
	var id RequestID

	if err := json.Unmarshal([]byte("42"), &id); err != nil {
		t.Fatalf("Unmarshal int failed: %v", err)
	}
	if id.isString {
		t.Error("expected int ID")
	}

	if err := json.Unmarshal([]byte(`"abc"`), &id); err != nil {
		t.Fatalf("Unmarshal string failed: %v", err)
	}
	if !id.isString {
		t.Error("expected string ID")
	}
}
