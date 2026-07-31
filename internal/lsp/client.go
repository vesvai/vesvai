package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Client struct {
	transport Transport

	mu           sync.RWMutex
	connected    bool
	nextID       int
	capabilities ServerCapabilities
	serverInfo   ServerInfo
	diagnostics  map[string][]Diagnostic
	docVersions  map[string]int
	rootURI      string
	notifHandler func(Notification)
}

type ClientOption func(*Client)

func WithRootURI(uri string) ClientOption {
	return func(c *Client) {
		c.rootURI = uri
	}
}

func WithNotificationHandler(handler func(Notification)) ClientOption {
	return func(c *Client) {
		c.notifHandler = handler
	}
}

func NewClient(transport Transport, opts ...ClientOption) *Client {
	c := &Client{
		transport:   transport,
		diagnostics: make(map[string][]Diagnostic),
		docVersions: make(map[string]int),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Connect(ctx context.Context) error {
	if err := c.transport.Start(ctx); err != nil {
		return fmt.Errorf("failed to start transport: %w", err)
	}

	c.transport.SetNotificationHandler(c.handleNotification)

	return c.initialize(ctx)
}

func (c *Client) initialize(ctx context.Context) error {
	params := InitializeParams{
		ProcessID: 0,
		RootURI:   c.rootURI,
		ClientInfo: ClientInfo{
			Name:    ClientName,
			Version: ClientVersion,
		},
		Capabilities: ClientCapabilities{
			TextDocument: &TextDocumentClientCapabilities{
				Completion: &CompletionClientCapabilities{
					CompletionItem: &CompletionItemCapabilities{
						SnippetSupport:          true,
						CommitCharactersSupport: true,
						DeprecatedSupport:       true,
						DocumentationFormat:     []string{"markdown", "plaintext"},
					},
				},
				Hover: &HoverClientCapabilities{
					ContentFormat: []string{"markdown", "plaintext"},
				},
				SignatureHelp: &SignatureHelpClientCapabilities{
					SignatureInformation: &SignatureInformationCapabilities{
						DocumentationFormat: []string{"markdown", "plaintext"},
					},
				},
				Definition: &DefinitionClientCapabilities{
					DynamicRegistration: false,
				},
				References: &ReferencesClientCapabilities{
					DynamicRegistration: false,
				},
				DocumentSymbol: &DocumentSymbolClientCapabilities{
					SymbolKind: &SymbolKindCapabilities{
						ValueSet: []int{
							SymbolKindFile, SymbolKindModule, SymbolKindNamespace,
							SymbolKindPackage, SymbolKindClass, SymbolKindMethod,
							SymbolKindProperty, SymbolKindField, SymbolKindConstructor,
							SymbolKindEnum, SymbolKindInterface, SymbolKindFunction,
							SymbolKindVariable, SymbolKindConstant, SymbolKindString,
							SymbolKindNumber, SymbolKindBoolean, SymbolKindArray,
							SymbolKindObject, SymbolKindKey, SymbolKindNull,
							SymbolKindEnumMember, SymbolKindStruct, SymbolKindEvent,
							SymbolKindOperator, SymbolKindTypeParameter,
						},
					},
				},
				CodeAction: &CodeActionClientCapabilities{
					DynamicRegistration: false,
				},
				PublishDiagnostics: &PublishDiagnosticsClientCapabilities{
					RelatedInformation: true,
				},
			},
			Workspace: &WorkspaceClientCapabilities{
				DidChangeWatchedFiles: &DidChangeWatchedFilesCapabilities{
					DynamicRegistration: false,
				},
			},
		},
	}

	id := c.nextRequestID()
	resp, err := c.transport.SendRequest(ctx, id, MethodInitialize, params)
	if err != nil {
		return fmt.Errorf("initialize request failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	var result InitializeResult
	if err := decodeResult(resp.Result, &result); err != nil {
		return fmt.Errorf("failed to decode initialize result: %w", err)
	}

	if err := c.transport.SendNotification(ctx, MethodInitialized, InitializedParams{}); err != nil {
		return fmt.Errorf("failed to send initialized notification: %w", err)
	}

	c.mu.Lock()
	c.connected = true
	c.capabilities = result.Capabilities
	c.serverInfo = result.ServerInfo
	c.mu.Unlock()

	return nil
}

func (c *Client) nextRequestID() int {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	c.mu.Unlock()
	return id
}

func (c *Client) handleNotification(notif Notification) {
	switch notif.Method {
	case MethodTextDocumentPublishDiagnostics:
		var params PublishDiagnosticsParams
		if err := decodeResult(notif.Params, &params); err != nil {
			return
		}
		c.mu.Lock()
		c.diagnostics[params.URI] = params.Diagnostics
		c.mu.Unlock()
	default:
		c.mu.RLock()
		handler := c.notifHandler
		c.mu.RUnlock()
		if handler != nil {
			handler(notif)
		}
	}
}

func (c *Client) Capabilities() ServerCapabilities {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.capabilities
}

func (c *Client) ServerInfo() ServerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverInfo
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

func (c *Client) sendRequest(ctx context.Context, method string, params any, target any) error {
	id := c.nextRequestID()
	resp, err := c.transport.SendRequest(ctx, id, method, params)
	if err != nil {
		return fmt.Errorf("%s request failed: %w", method, err)
	}
	if resp.Error != nil {
		return fmt.Errorf("%s error: %s", method, resp.Error.Message)
	}
	if target != nil {
		if err := decodeResult(resp.Result, target); err != nil {
			return fmt.Errorf("failed to decode %s result: %w", method, err)
		}
	}
	return nil
}

func (c *Client) sendNotification(ctx context.Context, method string, params any) error {
	return c.transport.SendNotification(ctx, method, params)
}

func (c *Client) Completion(ctx context.Context, uri string, position Position) (*CompletionList, error) {
	params := CompletionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}
	var result CompletionList
	if err := c.sendRequest(ctx, MethodTextDocumentCompletion, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Definition(ctx context.Context, uri string, position Position) ([]Location, error) {
	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}
	var raw json.RawMessage
	if err := c.sendRequest(ctx, MethodTextDocumentDefinition, params, &raw); err != nil {
		return nil, err
	}
	return decodeLocations(raw)
}

func (c *Client) References(ctx context.Context, uri string, position Position, includeDeclaration bool) ([]Location, error) {
	params := ReferenceParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
		Context: ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}
	var result []Location
	if err := c.sendRequest(ctx, MethodTextDocumentReferences, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Hover(ctx context.Context, uri string, position Position) (*HoverResult, error) {
	params := HoverParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}
	var result HoverResult
	if err := c.sendRequest(ctx, MethodTextDocumentHover, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DocumentSymbol(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	params := DocumentSymbolParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	var result []DocumentSymbol
	if err := c.sendRequest(ctx, MethodTextDocumentDocumentSymbol, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CodeAction(ctx context.Context, uri string, r Range, diagnostics []Diagnostic) ([]CodeAction, error) {
	params := CodeActionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Range:        r,
		Context: CodeActionContext{
			Diagnostics: diagnostics,
		},
	}
	var result []CodeAction
	if err := c.sendRequest(ctx, MethodTextDocumentCodeAction, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Diagnostics(_ context.Context, uri string) ([]Diagnostic, error) {
	c.mu.RLock()
	diags := c.diagnostics[uri]
	c.mu.RUnlock()
	return diags, nil
}

func (c *Client) Format(ctx context.Context, uri string, options FormattingOptions) ([]TextEdit, error) {
	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Options      FormattingOptions      `json:"options"`
	}{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Options:      options,
	}
	var result []TextEdit
	if err := c.sendRequest(ctx, "textDocument/formatting", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) Rename(ctx context.Context, uri string, position Position, newName string) (*WorkspaceEdit, error) {
	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		NewName      string                 `json:"newName"`
	}{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
		NewName:      newName,
	}
	var result WorkspaceEdit
	if err := c.sendRequest(ctx, "textDocument/rename", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) SignatureHelp(ctx context.Context, uri string, position Position) (*SignatureHelp, error) {
	params := SignatureHelpParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}
	var result SignatureHelp
	if err := c.sendRequest(ctx, MethodTextDocumentSignatureHelp, params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DidOpen(ctx context.Context, uri, languageId, text string) error {
	c.mu.Lock()
	c.docVersions[uri] = 1
	c.mu.Unlock()

	params := DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        uri,
			LanguageID: languageId,
			Version:    1,
			Text:       text,
		},
	}
	return c.sendNotification(ctx, MethodTextDocumentDidOpen, params)
}

func (c *Client) DidChange(ctx context.Context, uri string, changes []TextDocumentContentChangeEvent) error {
	c.mu.Lock()
	c.docVersions[uri]++
	version := c.docVersions[uri]
	c.mu.Unlock()

	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: version,
		},
		ContentChanges: changes,
	}
	return c.sendNotification(ctx, MethodTextDocumentDidChange, params)
}

func (c *Client) DidClose(ctx context.Context, uri string) error {
	c.mu.Lock()
	delete(c.docVersions, uri)
	delete(c.diagnostics, uri)
	c.mu.Unlock()

	params := DidCloseTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	return c.sendNotification(ctx, MethodTextDocumentDidClose, params)
}

func (c *Client) DidSave(ctx context.Context, uri string) error {
	params := DidSaveTextDocumentParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}
	return c.sendNotification(ctx, MethodTextDocumentDidSave, params)
}

func (c *Client) Close() error {
	if c.IsConnected() {
		ctx := context.Background()
		_ = c.sendRequest(ctx, MethodShutdown, nil, nil)
		_ = c.sendNotification(ctx, MethodExit, nil)
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
	}
	return c.transport.Close()
}

type WorkspaceEdit struct {
	Changes map[string][]TextEdit `json:"changes,omitempty"`
}

type FormattingOptions struct {
	TabSize      int  `json:"tabSize"`
	InsertSpaces bool `json:"insertSpaces"`
}

func decodeResult(raw any, target any) error {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	return json.Unmarshal(data, target)
}

func decodeLocations(raw json.RawMessage) ([]Location, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var locations []Location
	if err := json.Unmarshal(raw, &locations); err == nil {
		return locations, nil
	}

	var loc Location
	if err := json.Unmarshal(raw, &loc); err == nil {
		return []Location{loc}, nil
	}

	var links []LocationLink
	if err := json.Unmarshal(raw, &links); err == nil {
		result := make([]Location, 0, len(links))
		for _, link := range links {
			result = append(result, Location{
				URI:   link.TargetURI,
				Range: link.TargetSelectionRange,
			})
		}
		return result, nil
	}

	return nil, fmt.Errorf("unexpected definition response type")
}
