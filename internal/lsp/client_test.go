package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

type mockTransport struct {
	mu sync.Mutex

	handlers map[string]func(id int, params any) (*Response, error)

	notifications []Notification
	started       bool
	closed        bool

	notifHandler func(Notification)
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		handlers: make(map[string]func(id int, params any) (*Response, error)),
	}
}

func (m *mockTransport) Start(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.started = true
	return nil
}

func (m *mockTransport) SendRequest(_ context.Context, id int, method string, params any) (*Response, error) {
	m.mu.Lock()
	handler, ok := m.handlers[method]
	m.mu.Unlock()

	if !ok {
		return &Response{
			JSONRPC: jsonrpcVersion,
			Error: &RPCError{
				Code:    ErrCodeMethodNotFound,
				Message: fmt.Sprintf("method not found: %s", method),
			},
		}, nil
	}

	return handler(id, params)
}

func (m *mockTransport) SendNotification(_ context.Context, method string, params any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, Notification{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	})
	return nil
}

func (m *mockTransport) ReadMessage(_ context.Context) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockTransport) SetNotificationHandler(handler func(notification Notification)) {
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

func (m *mockTransport) setHandler(method string, handler func(id int, params any) (*Response, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[method] = handler
}

func (m *mockTransport) getNotifications() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Notification, len(m.notifications))
	copy(out, m.notifications)
	return out
}

func (m *mockTransport) lastNotification() *Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.notifications) == 0 {
		return nil
	}
	return &m.notifications[len(m.notifications)-1]
}

func newConnectedClient(t *testing.T) (*Client, *mockTransport) {
	t.Helper()
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return &Response{
			JSONRPC: jsonrpcVersion,
			ID:      IntID(int64(id)),
			Result: InitializeResult{
				Capabilities: ServerCapabilities{
					CompletionProvider: &CompletionOptions{
						TriggerCharacters: []string{".", "("},
					},
					HoverProvider: true,
				},
				ServerInfo: ServerInfo{
					Name:    "test-server",
					Version: "1.0.0",
				},
			},
		}, nil
	})

	client := NewClient(mock)
	ctx := context.Background()
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	return client, mock
}

func okResponse(id int, result any) *Response {
	return &Response{
		JSONRPC: jsonrpcVersion,
		ID:      IntID(int64(id)),
		Result:  result,
	}
}

func errResponse(id int, code int, msg string) *Response {
	return &Response{
		JSONRPC: jsonrpcVersion,
		ID:      IntID(int64(id)),
		Error: &RPCError{
			Code:    code,
			Message: msg,
		},
	}
}

func TestClient_Connect(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			Capabilities: ServerCapabilities{
				CompletionProvider: &CompletionOptions{
					TriggerCharacters: []string{"."},
				},
				HoverProvider: true,
			},
			ServerInfo: ServerInfo{
				Name:    "test-server",
				Version: "1.0.0",
			},
		}), nil
	})

	client := NewClient(mock, WithRootURI("file:///workspace"))
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if !mock.started {
		t.Fatal("transport was not started")
	}

	if !client.IsConnected() {
		t.Fatal("client should be connected")
	}

	info := client.ServerInfo()
	if info.Name != "test-server" {
		t.Errorf("expected server name 'test-server', got %q", info.Name)
	}
	if info.Version != "1.0.0" {
		t.Errorf("expected server version '1.0.0', got %q", info.Version)
	}

	caps := client.Capabilities()
	if caps.CompletionProvider == nil {
		t.Fatal("expected completion provider")
	}
	if caps.CompletionProvider.TriggerCharacters[0] != "." {
		t.Errorf("expected trigger character '.', got %q", caps.CompletionProvider.TriggerCharacters[0])
	}
}

func TestClient_Connect_TransportStartError(t *testing.T) {
	client := NewClient(&failingTransport{err: fmt.Errorf("connection refused")})
	ctx := context.Background()

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected error from Connect")
	}
}

func TestClient_Connect_InitializeError(t *testing.T) {
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return errResponse(id, ErrCodeInternalError, "server initialization failed"), nil
	})

	client := NewClient(mock)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected error from Connect")
	}
	if err.Error() != "initialize error: server initialization failed" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestClient_Completion(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentCompletion, func(id int, params any) (*Response, error) {
		return okResponse(id, CompletionList{
			IsIncomplete: false,
			Items: []CompletionItem{
				{
					Label:      "Hello",
					Kind:       CompletionItemKindFunction,
					Detail:     "func Hello() string",
					InsertText: "Hello()",
				},
				{
					Label:      "World",
					Kind:       CompletionItemKindVariable,
					Detail:     "string",
					InsertText: "World",
				},
			},
		}), nil
	})

	result, err := client.Completion(context.Background(), "file:///test.go", Position{Line: 10, Character: 5})
	if err != nil {
		t.Fatalf("Completion failed: %v", err)
	}

	if result.IsIncomplete {
		t.Error("expected IsIncomplete to be false")
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 completion items, got %d", len(result.Items))
	}
	if result.Items[0].Label != "Hello" {
		t.Errorf("expected first item label 'Hello', got %q", result.Items[0].Label)
	}
	if result.Items[0].Kind != CompletionItemKindFunction {
		t.Errorf("expected kind %d (Function), got %d", CompletionItemKindFunction, result.Items[0].Kind)
	}
	if result.Items[1].Label != "World" {
		t.Errorf("expected second item label 'World', got %q", result.Items[1].Label)
	}
}

func TestClient_Definition_SingleLocation(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentDefinition, func(id int, params any) (*Response, error) {
		return okResponse(id, Location{
			URI: "file:///other.go",
			Range: Range{
				Start: Position{Line: 5, Character: 0},
				End:   Position{Line: 5, Character: 10},
			},
		}), nil
	})

	locations, err := client.Definition(context.Background(), "file:///test.go", Position{Line: 10, Character: 5})
	if err != nil {
		t.Fatalf("Definition failed: %v", err)
	}

	if len(locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locations))
	}
	if locations[0].URI != "file:///other.go" {
		t.Errorf("expected URI 'file:///other.go', got %q", locations[0].URI)
	}
	if locations[0].Range.Start.Line != 5 {
		t.Errorf("expected start line 5, got %d", locations[0].Range.Start.Line)
	}
}

func TestClient_Definition_MultipleLocations(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentDefinition, func(id int, params any) (*Response, error) {
		return okResponse(id, []Location{
			{
				URI:   "file:///a.go",
				Range: Range{Start: Position{Line: 1, Character: 0}, End: Position{Line: 1, Character: 5}},
			},
			{
				URI:   "file:///b.go",
				Range: Range{Start: Position{Line: 2, Character: 0}, End: Position{Line: 2, Character: 5}},
			},
		}), nil
	})

	locations, err := client.Definition(context.Background(), "file:///test.go", Position{Line: 0, Character: 0})
	if err != nil {
		t.Fatalf("Definition failed: %v", err)
	}

	if len(locations) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(locations))
	}
	if locations[0].URI != "file:///a.go" {
		t.Errorf("expected first URI 'file:///a.go', got %q", locations[0].URI)
	}
	if locations[1].URI != "file:///b.go" {
		t.Errorf("expected second URI 'file:///b.go', got %q", locations[1].URI)
	}
}

func TestDecodeLocations_LocationLinks(t *testing.T) {
	raw := json.RawMessage(`[{"targetUri":"file:///target.go","targetRange":{"start":{"line":10,"character":0},"end":{"line":10,"character":20}},"targetSelectionRange":{"start":{"line":10,"character":5},"end":{"line":10,"character":15}}}]`)
	locations, err := decodeLocations(raw)
	if err != nil {
		t.Fatalf("decodeLocations failed: %v", err)
	}

	if len(locations) != 1 {
		t.Fatalf("expected 1 location, got %d", len(locations))
	}
}

func TestDecodeLocations_Empty(t *testing.T) {
	locations, err := decodeLocations(json.RawMessage{})
	if err != nil {
		t.Fatalf("decodeLocations failed: %v", err)
	}
	if locations != nil {
		t.Errorf("expected nil, got %v", locations)
	}
}

func TestClient_References(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentReferences, func(id int, params any) (*Response, error) {
		return okResponse(id, []Location{
			{
				URI:   "file:///a.go",
				Range: Range{Start: Position{Line: 3, Character: 2}, End: Position{Line: 3, Character: 7}},
			},
			{
				URI:   "file:///b.go",
				Range: Range{Start: Position{Line: 10, Character: 5}, End: Position{Line: 10, Character: 10}},
			},
			{
				URI:   "file:///test.go",
				Range: Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 5}},
			},
		}), nil
	})

	locations, err := client.References(context.Background(), "file:///test.go", Position{Line: 0, Character: 0}, true)
	if err != nil {
		t.Fatalf("References failed: %v", err)
	}

	if len(locations) != 3 {
		t.Fatalf("expected 3 references, got %d", len(locations))
	}
	if locations[0].URI != "file:///a.go" {
		t.Errorf("expected first URI 'file:///a.go', got %q", locations[0].URI)
	}
}

func TestClient_Hover(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentHover, func(id int, params any) (*Response, error) {
		return okResponse(id, HoverResult{
			Contents: MarkedContent{
				Kind:  "markdown",
				Value: "**func** Hello()\n\nReturns a greeting",
			},
			Range: &Range{
				Start: Position{Line: 5, Character: 0},
				End:   Position{Line: 5, Character: 5},
			},
		}), nil
	})

	result, err := client.Hover(context.Background(), "file:///test.go", Position{Line: 5, Character: 2})
	if err != nil {
		t.Fatalf("Hover failed: %v", err)
	}

	if result.Contents.Kind != "markdown" {
		t.Errorf("expected content kind 'markdown', got %q", result.Contents.Kind)
	}
	if result.Contents.Value != "**func** Hello()\n\nReturns a greeting" {
		t.Errorf("unexpected hover value: %q", result.Contents.Value)
	}
	if result.Range == nil {
		t.Fatal("expected range to be set")
	}
	if result.Range.Start.Line != 5 {
		t.Errorf("expected range start line 5, got %d", result.Range.Start.Line)
	}
}

func TestClient_DocumentSymbol(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentDocumentSymbol, func(id int, params any) (*Response, error) {
		return okResponse(id, []DocumentSymbol{
			{
				Name:           "MyStruct",
				Detail:         "struct",
				Kind:           SymbolKindStruct,
				Range:          Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 20, Character: 0}},
				SelectionRange: Range{Start: Position{Line: 0, Character: 7}, End: Position{Line: 0, Character: 14}},
				Children: []DocumentSymbol{
					{
						Name:           "Field1",
						Detail:         "string",
						Kind:           SymbolKindField,
						Range:          Range{Start: Position{Line: 1, Character: 1}, End: Position{Line: 1, Character: 15}},
						SelectionRange: Range{Start: Position{Line: 1, Character: 1}, End: Position{Line: 1, Character: 7}},
					},
				},
			},
			{
				Name:           "DoStuff",
				Detail:         "func DoStuff()",
				Kind:           SymbolKindFunction,
				Range:          Range{Start: Position{Line: 22, Character: 0}, End: Position{Line: 30, Character: 0}},
				SelectionRange: Range{Start: Position{Line: 22, Character: 5}, End: Position{Line: 22, Character: 12}},
			},
		}), nil
	})

	symbols, err := client.DocumentSymbol(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DocumentSymbol failed: %v", err)
	}

	if len(symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(symbols))
	}
	if symbols[0].Name != "MyStruct" {
		t.Errorf("expected first symbol 'MyStruct', got %q", symbols[0].Name)
	}
	if symbols[0].Kind != SymbolKindStruct {
		t.Errorf("expected kind %d (Struct), got %d", SymbolKindStruct, symbols[0].Kind)
	}
	if len(symbols[0].Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(symbols[0].Children))
	}
	if symbols[0].Children[0].Name != "Field1" {
		t.Errorf("expected child 'Field1', got %q", symbols[0].Children[0].Name)
	}
	if symbols[1].Name != "DoStuff" {
		t.Errorf("expected second symbol 'DoStuff', got %q", symbols[1].Name)
	}
}

func TestClient_CodeAction(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentCodeAction, func(id int, params any) (*Response, error) {
		return okResponse(id, []CodeAction{
			{
				Title:       "Remove unused import",
				Kind:        "quickfix",
				IsPreferred: true,
				Diagnostics: []Diagnostic{
					{
						Range:    Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 20}},
						Severity: DiagnosticSeverityWarning,
						Message:  "imported and not used: fmt",
					},
				},
				Command: &Command{
					Title:     "Remove import",
					Command:   "editor.action.removeImport",
					Arguments: []any{"fmt"},
				},
			},
		}), nil
	})

	diags := []Diagnostic{
		{
			Range:    Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 20}},
			Severity: DiagnosticSeverityWarning,
			Message:  "imported and not used: fmt",
		},
	}

	actions, err := client.CodeAction(context.Background(), "file:///test.go",
		Range{Start: Position{Line: 0, Character: 0}, End: Position{Line: 0, Character: 20}},
		diags,
	)
	if err != nil {
		t.Fatalf("CodeAction failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 code action, got %d", len(actions))
	}
	if actions[0].Title != "Remove unused import" {
		t.Errorf("expected title 'Remove unused import', got %q", actions[0].Title)
	}
	if actions[0].Kind != "quickfix" {
		t.Errorf("expected kind 'quickfix', got %q", actions[0].Kind)
	}
	if !actions[0].IsPreferred {
		t.Error("expected IsPreferred to be true")
	}
	if actions[0].Command == nil {
		t.Fatal("expected command to be set")
	}
	if actions[0].Command.Command != "editor.action.removeImport" {
		t.Errorf("expected command 'editor.action.removeImport', got %q", actions[0].Command.Command)
	}
}

func TestClient_Diagnostics(t *testing.T) {
	client, mock := newConnectedClient(t)

	diags := []Diagnostic{
		{
			Range:    Range{Start: Position{Line: 5, Character: 0}, End: Position{Line: 5, Character: 10}},
			Severity: DiagnosticSeverityError,
			Message:  "undefined: foo",
			Source:   "go",
		},
		{
			Range:    Range{Start: Position{Line: 10, Character: 0}, End: Position{Line: 10, Character: 5}},
			Severity: DiagnosticSeverityWarning,
			Message:  "unused variable: bar",
			Source:   "go",
		},
	}

	mock.mu.Lock()
	handler := mock.notifHandler
	mock.mu.Unlock()

	if handler != nil {
		handler(Notification{
			JSONRPC: jsonrpcVersion,
			Method:  MethodTextDocumentPublishDiagnostics,
			Params: PublishDiagnosticsParams{
				URI:         "file:///test.go",
				Diagnostics: diags,
			},
		})
	}

	result, err := client.Diagnostics(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("Diagnostics failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(result))
	}
	if result[0].Severity != DiagnosticSeverityError {
		t.Errorf("expected severity Error (%d), got %d", DiagnosticSeverityError, result[0].Severity)
	}
	if result[0].Message != "undefined: foo" {
		t.Errorf("expected message 'undefined: foo', got %q", result[0].Message)
	}
	if result[1].Severity != DiagnosticSeverityWarning {
		t.Errorf("expected severity Warning (%d), got %d", DiagnosticSeverityWarning, result[1].Severity)
	}
}

func TestClient_Diagnostics_Empty(t *testing.T) {
	client, _ := newConnectedClient(t)

	result, err := client.Diagnostics(context.Background(), "file:///nonexistent.go")
	if err != nil {
		t.Fatalf("Diagnostics failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(result))
	}
}

func TestClient_Format(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler("textDocument/formatting", func(id int, params any) (*Response, error) {
		return okResponse(id, []TextEdit{
			{
				Range: Range{
					Start: Position{Line: 0, Character: 0},
					End:   Position{Line: 0, Character: 5},
				},
				NewText: "updated",
			},
		}), nil
	})

	edits, err := client.Format(context.Background(), "file:///test.go", FormattingOptions{
		TabSize:      4,
		InsertSpaces: true,
	})
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	if len(edits) != 1 {
		t.Fatalf("expected 1 text edit, got %d", len(edits))
	}
	if edits[0].NewText != "updated" {
		t.Errorf("expected newText 'updated', got %q", edits[0].NewText)
	}
}

func TestClient_Rename(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler("textDocument/rename", func(id int, params any) (*Response, error) {
		return okResponse(id, WorkspaceEdit{
			Changes: map[string][]TextEdit{
				"file:///test.go": {
					{
						Range:   Range{Start: Position{Line: 5, Character: 2}, End: Position{Line: 5, Character: 7}},
						NewText: "NewName",
					},
					{
						Range:   Range{Start: Position{Line: 10, Character: 5}, End: Position{Line: 10, Character: 10}},
						NewText: "NewName",
					},
				},
			},
		}), nil
	})

	edit, err := client.Rename(context.Background(), "file:///test.go", Position{Line: 5, Character: 2}, "NewName")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	if len(edit.Changes) != 1 {
		t.Fatalf("expected 1 file in changes, got %d", len(edit.Changes))
	}

	edits := edit.Changes["file:///test.go"]
	if len(edits) != 2 {
		t.Fatalf("expected 2 edits, got %d", len(edits))
	}
	if edits[0].NewText != "NewName" {
		t.Errorf("expected rename text 'NewName', got %q", edits[0].NewText)
	}
}

func TestClient_SignatureHelp(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentSignatureHelp, func(id int, params any) (*Response, error) {
		return okResponse(id, SignatureHelp{
			Signatures: []SignatureInformation{
				{
					Label:         "func Hello(name string) string",
					Documentation: "Returns a greeting for the given name",
					Parameters: []ParameterInformation{
						{
							Label:         "name string",
							Documentation: "The name to greet",
						},
					},
				},
			},
			ActiveSignature: 0,
			ActiveParameter: 0,
		}), nil
	})

	result, err := client.SignatureHelp(context.Background(), "file:///test.go", Position{Line: 10, Character: 15})
	if err != nil {
		t.Fatalf("SignatureHelp failed: %v", err)
	}

	if len(result.Signatures) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(result.Signatures))
	}
	if result.Signatures[0].Label != "func Hello(name string) string" {
		t.Errorf("unexpected signature label: %q", result.Signatures[0].Label)
	}
	if result.ActiveSignature != 0 {
		t.Errorf("expected active signature 0, got %d", result.ActiveSignature)
	}
	if result.ActiveParameter != 0 {
		t.Errorf("expected active parameter 0, got %d", result.ActiveParameter)
	}
	if len(result.Signatures[0].Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(result.Signatures[0].Parameters))
	}
}

func TestClient_DidOpen(t *testing.T) {
	client, mock := newConnectedClient(t)

	err := client.DidOpen(context.Background(), "file:///test.go", "go", "package main\n\nfunc main() {}")
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	notifications := mock.getNotifications()
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications (initialized + didOpen), got %d", len(notifications))
	}
	if notifications[1].Method != MethodTextDocumentDidOpen {
		t.Errorf("expected method %q, got %q", MethodTextDocumentDidOpen, notifications[1].Method)
	}
}

func TestClient_DidOpen_Params(t *testing.T) {
	client, mock := newConnectedClient(t)

	err := client.DidOpen(context.Background(), "file:///test.go", "go", "package main")
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	notif := mock.lastNotification()
	if notif == nil {
		t.Fatal("expected notification")
	}

	data, err := json.Marshal(notif.Params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}

	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(data, &params); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}

	if params.TextDocument.URI != "file:///test.go" {
		t.Errorf("expected URI 'file:///test.go', got %q", params.TextDocument.URI)
	}
	if params.TextDocument.LanguageID != "go" {
		t.Errorf("expected language ID 'go', got %q", params.TextDocument.LanguageID)
	}
	if params.TextDocument.Version != 1 {
		t.Errorf("expected version 1, got %d", params.TextDocument.Version)
	}
	if params.TextDocument.Text != "package main" {
		t.Errorf("expected text 'package main', got %q", params.TextDocument.Text)
	}
}

func TestClient_DidChange(t *testing.T) {
	client, mock := newConnectedClient(t)

	_ = client.DidOpen(context.Background(), "file:///test.go", "go", "initial")

	changes := []TextDocumentContentChangeEvent{
		{
			Text: "updated content",
		},
	}
	err := client.DidChange(context.Background(), "file:///test.go", changes)
	if err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	notifications := mock.getNotifications()
	if len(notifications) != 3 {
		t.Fatalf("expected 3 notifications, got %d", len(notifications))
	}
	if notifications[2].Method != MethodTextDocumentDidChange {
		t.Errorf("expected method %q, got %q", MethodTextDocumentDidChange, notifications[2].Method)
	}

	data, err := json.Marshal(notifications[2].Params)
	if err != nil {
		t.Fatalf("failed to marshal params: %v", err)
	}
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(data, &params); err != nil {
		t.Fatalf("failed to unmarshal params: %v", err)
	}
	if params.TextDocument.Version != 2 {
		t.Errorf("expected version 2, got %d", params.TextDocument.Version)
	}
}

func TestClient_DidChange_MultipleChanges(t *testing.T) {
	client, mock := newConnectedClient(t)

	_ = client.DidOpen(context.Background(), "file:///test.go", "go", "v1")

	_ = client.DidChange(context.Background(), "file:///test.go", []TextDocumentContentChangeEvent{
		{Text: "v2"},
	})
	_ = client.DidChange(context.Background(), "file:///test.go", []TextDocumentContentChangeEvent{
		{Text: "v3"},
	})

	notifications := mock.getNotifications()
	if len(notifications) != 4 {
		t.Fatalf("expected 4 notifications, got %d", len(notifications))
	}

	data, _ := json.Marshal(notifications[3].Params)
	var params DidChangeTextDocumentParams
	json.Unmarshal(data, &params)
	if params.TextDocument.Version != 3 {
		t.Errorf("expected version 3, got %d", params.TextDocument.Version)
	}
}

func TestClient_DidClose(t *testing.T) {
	client, mock := newConnectedClient(t)

	_ = client.DidOpen(context.Background(), "file:///test.go", "go", "content")
	err := client.DidClose(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DidClose failed: %v", err)
	}

	notifications := mock.getNotifications()
	if len(notifications) != 3 {
		t.Fatalf("expected 3 notifications, got %d", len(notifications))
	}
	if notifications[2].Method != MethodTextDocumentDidClose {
		t.Errorf("expected method %q, got %q", MethodTextDocumentDidClose, notifications[2].Method)
	}
}

func TestClient_DidClose_ClearsDiagnostics(t *testing.T) {
	client, mock := newConnectedClient(t)

	_ = client.DidOpen(context.Background(), "file:///test.go", "go", "content")

	mock.mu.Lock()
	handler := mock.notifHandler
	mock.mu.Unlock()

	handler(Notification{
		Method: MethodTextDocumentPublishDiagnostics,
		Params: PublishDiagnosticsParams{
			URI: "file:///test.go",
			Diagnostics: []Diagnostic{
				{Range: Range{Start: Position{}, End: Position{}}, Message: "error"},
			},
		},
	})

	diags, _ := client.Diagnostics(context.Background(), "file:///test.go")
	if len(diags) != 1 {
		t.Fatal("expected 1 diagnostic before close")
	}

	_ = client.DidClose(context.Background(), "file:///test.go")

	diags, _ = client.Diagnostics(context.Background(), "file:///test.go")
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics after close, got %d", len(diags))
	}
}

func TestClient_DidSave(t *testing.T) {
	client, mock := newConnectedClient(t)

	err := client.DidSave(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DidSave failed: %v", err)
	}

	notifications := mock.getNotifications()
	if len(notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifications))
	}
	if notifications[1].Method != MethodTextDocumentDidSave {
		t.Errorf("expected method %q, got %q", MethodTextDocumentDidSave, notifications[1].Method)
	}
}

func TestClient_Close(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodShutdown, func(id int, params any) (*Response, error) {
		return okResponse(id, nil), nil
	})

	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !mock.closed {
		t.Error("transport was not closed")
	}
	if client.IsConnected() {
		t.Error("client should not be connected after Close")
	}

	notifications := mock.getNotifications()
	foundExit := false
	for _, n := range notifications {
		if n.Method == MethodExit {
			foundExit = true
		}
	}
	if !foundExit {
		t.Error("expected exit notification to be sent")
	}
}

func TestClient_Close_NotConnected(t *testing.T) {
	mock := newMockTransport()
	client := NewClient(mock)

	err := client.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if !mock.closed {
		t.Error("transport was not closed")
	}

	notifications := mock.getNotifications()
	if len(notifications) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(notifications))
	}
}

func TestClient_Completion_ServerError(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentCompletion, func(id int, params any) (*Response, error) {
		return errResponse(id, -32603, "internal error"), nil
	})

	_, err := client.Completion(context.Background(), "file:///test.go", Position{})
	if err == nil {
		t.Fatal("expected error from Completion")
	}
}

func TestClient_Definition_ServerError(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentDefinition, func(id int, params any) (*Response, error) {
		return errResponse(id, -32603, "no definition found"), nil
	})

	_, err := client.Definition(context.Background(), "file:///test.go", Position{})
	if err == nil {
		t.Fatal("expected error from Definition")
	}
}

func TestClient_Hover_ServerError(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentHover, func(id int, params any) (*Response, error) {
		return errResponse(id, -32603, "no hover info"), nil
	})

	_, err := client.Hover(context.Background(), "file:///test.go", Position{})
	if err == nil {
		t.Fatal("expected error from Hover")
	}
}

func TestClient_References_ServerError(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler(MethodTextDocumentReferences, func(id int, params any) (*Response, error) {
		return errResponse(id, -32603, "references failed"), nil
	})

	_, err := client.References(context.Background(), "file:///test.go", Position{}, true)
	if err == nil {
		t.Fatal("expected error from References")
	}
}

func TestClient_Rename_ServerError(t *testing.T) {
	client, mock := newConnectedClient(t)

	mock.setHandler("textDocument/rename", func(id int, params any) (*Response, error) {
		return errResponse(id, -32603, "rename not supported"), nil
	})

	_, err := client.Rename(context.Background(), "file:///test.go", Position{}, "new")
	if err == nil {
		t.Fatal("expected error from Rename")
	}
}

func TestClient_NotificationHandler(t *testing.T) {
	mock := newMockTransport()

	var received []Notification
	handler := func(n Notification) {
		received = append(received, n)
	}

	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
		}), nil
	})

	client := NewClient(mock, WithNotificationHandler(handler))
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	mock.mu.Lock()
	notifHandler := mock.notifHandler
	mock.mu.Unlock()

	notifHandler(Notification{
		Method: "custom/notification",
		Params: map[string]string{"key": "value"},
	})

	if len(received) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(received))
	}
	if received[0].Method != "custom/notification" {
		t.Errorf("expected method 'custom/notification', got %q", received[0].Method)
	}
}

func TestClient_PublishDiagnostics_NotForwardedToHandler(t *testing.T) {
	mock := newMockTransport()

	var received []Notification
	handler := func(n Notification) {
		received = append(received, n)
	}

	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
		}), nil
	})

	client := NewClient(mock, WithNotificationHandler(handler))
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	mock.mu.Lock()
	notifHandler := mock.notifHandler
	mock.mu.Unlock()

	notifHandler(Notification{
		Method: MethodTextDocumentPublishDiagnostics,
		Params: PublishDiagnosticsParams{
			URI: "file:///test.go",
			Diagnostics: []Diagnostic{
				{Range: Range{}, Message: "error"},
			},
		},
	})

	if len(received) != 0 {
		t.Errorf("expected 0 forwarded notifications, got %d", len(received))
	}
}

func TestClient_WithRootURI(t *testing.T) {
	var capturedParams any
	mock := newMockTransport()

	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		capturedParams = params
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "test", Version: "1.0.0"},
		}), nil
	})

	client := NewClient(mock, WithRootURI("file:///my-project"))
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	data, _ := json.Marshal(capturedParams)
	var params InitializeParams
	json.Unmarshal(data, &params)

	if params.RootURI != "file:///my-project" {
		t.Errorf("expected root URI 'file:///my-project', got %q", params.RootURI)
	}
}

func TestClient_RequestIDIncrementing(t *testing.T) {
	client, mock := newConnectedClient(t)

	var ids []int
	mock.setHandler(MethodTextDocumentCompletion, func(id int, params any) (*Response, error) {
		ids = append(ids, id)
		return okResponse(id, CompletionList{Items: []CompletionItem{}}), nil
	})

	_, _ = client.Completion(context.Background(), "file:///test.go", Position{})
	_, _ = client.Completion(context.Background(), "file:///test.go", Position{})
	_, _ = client.Completion(context.Background(), "file:///test.go", Position{})

	if len(ids) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(ids))
	}

	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("IDs not incrementing: %d <= %d", ids[i], ids[i-1])
		}
	}
}

type failingTransport struct {
	err error
}

func (f *failingTransport) Start(_ context.Context) error {
	return f.err
}

func (f *failingTransport) SendRequest(_ context.Context, _ int, _ string, _ any) (*Response, error) {
	return nil, f.err
}

func (f *failingTransport) SendNotification(_ context.Context, _ string, _ any) error {
	return f.err
}

func (f *failingTransport) ReadMessage(_ context.Context) (any, error) {
	return nil, f.err
}

func (f *failingTransport) SetNotificationHandler(_ func(Notification)) {}

func (f *failingTransport) Close() error {
	return nil
}
