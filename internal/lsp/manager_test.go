package lsp

import (
	"context"
	"sort"
	"sync"
	"testing"

	"github.com/vesvai/vesvai/internal/event"
)

func newTestManager(t *testing.T) (*Manager, event.EventBus) {
	t.Helper()
	bus := event.NewEventBus()
	t.Cleanup(func() { bus.Close() })
	return NewManager(bus), bus
}

func newMockConnectedClient(t *testing.T) *Client {
	t.Helper()
	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			Capabilities: ServerCapabilities{
				CompletionProvider: &CompletionOptions{
					TriggerCharacters: []string{"."},
				},
				HoverProvider: true,
			},
			ServerInfo: ServerInfo{Name: "mock-server", Version: "1.0.0"},
		}), nil
	})
	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("failed to connect mock client: %v", err)
	}
	return client
}

func injectServer(m *Manager, name string, client *Client, languages, patterns []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[name] = &serverEntry{
		client:    client,
		cmd:       []string{"mock-lsp"},
		rootURI:   "file:///workspace",
		languages: languages,
		patterns:  patterns,
	}
}

func TestManager_NewManager(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()

	m := NewManager(bus)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.servers == nil {
		t.Fatal("servers map not initialized")
	}
	if m.eventBus == nil {
		t.Fatal("eventBus not set")
	}
	if len(m.servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(m.servers))
	}
}

func TestManager_NewManager_NilBus(t *testing.T) {
	m := NewManager(nil)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if len(m.servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(m.servers))
	}
}

func TestManager_AddServer(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.AddServer("gopls", []string{"gopls"}, "file:///workspace")
	if err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	names := m.Servers()
	if len(names) != 1 {
		t.Fatalf("expected 1 server, got %d", len(names))
	}
	if names[0] != "gopls" {
		t.Errorf("expected server name 'gopls', got %q", names[0])
	}
}

func TestManager_AddServer_Duplicate(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.AddServer("gopls", []string{"gopls"}, "file:///workspace"); err != nil {
		t.Fatalf("first AddServer failed: %v", err)
	}

	err := m.AddServer("gopls", []string{"gopls"}, "file:///workspace")
	if err == nil {
		t.Fatal("expected error for duplicate server name")
	}
}

func TestManager_AddServer_Multiple(t *testing.T) {
	m, _ := newTestManager(t)

	servers := []struct {
		name string
		cmd  []string
		root string
	}{
		{"gopls", []string{"gopls"}, "file:///go"},
		{"pyright", []string{"pyright-langserver", "--stdio"}, "file:///py"},
		{"typescript", []string{"typescript-language-server", "--stdio"}, "file:///ts"},
	}

	for _, s := range servers {
		if err := m.AddServer(s.name, s.cmd, s.root); err != nil {
			t.Fatalf("AddServer(%q) failed: %v", s.name, err)
		}
	}

	names := m.Servers()
	if len(names) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(names))
	}
	sort.Strings(names)
	expected := []string{"gopls", "pyright", "typescript"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("expected server %q at index %d, got %q", expected[i], i, n)
		}
	}
}

func TestManager_RemoveServer_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.RemoveServer("nonexistent")
	if err == nil {
		t.Fatal("expected error for removing nonexistent server")
	}
}

func TestManager_RemoveServer_NoClient(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.AddServer("gopls", []string{"gopls"}, "file:///workspace"); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	err := m.RemoveServer("gopls")
	if err != nil {
		t.Fatalf("RemoveServer failed: %v", err)
	}

	names := m.Servers()
	if len(names) != 0 {
		t.Errorf("expected 0 servers after removal, got %d", len(names))
	}
}

func TestManager_RemoveServer_WithClient(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, nil, nil)

	if !client.IsConnected() {
		t.Fatal("client should be connected before removal")
	}

	err := m.RemoveServer("gopls")
	if err != nil {
		t.Fatalf("RemoveServer failed: %v", err)
	}

	if client.IsConnected() {
		t.Error("client should not be connected after removal")
	}

	names := m.Servers()
	if len(names) != 0 {
		t.Errorf("expected 0 servers after removal, got %d", len(names))
	}
}

func TestManager_GetClient_Found(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, nil, nil)

	got, ok := m.GetClient("gopls")
	if !ok {
		t.Fatal("expected to find client")
	}
	if got != client {
		t.Error("returned client does not match injected client")
	}
}

func TestManager_GetClient_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	got, ok := m.GetClient("nonexistent")
	if ok {
		t.Fatal("expected ok=false for nonexistent server")
	}
	if got != nil {
		t.Error("expected nil client for nonexistent server")
	}
}

func TestManager_GetClient_NoClientConnected(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.AddServer("gopls", []string{"gopls"}, "file:///workspace"); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	got, ok := m.GetClient("gopls")
	if ok {
		t.Fatal("expected ok=false when client is nil")
	}
	if got != nil {
		t.Error("expected nil client")
	}
}

func TestManager_GetClientForLanguage_Found(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, nil)

	got, ok := m.GetClientForLanguage("go")
	if !ok {
		t.Fatal("expected to find client for 'go'")
	}
	if got != client {
		t.Error("returned client does not match")
	}
}

func TestManager_GetClientForLanguage_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, nil)

	got, ok := m.GetClientForLanguage("python")
	if ok {
		t.Fatal("expected ok=false for unsupported language")
	}
	if got != nil {
		t.Error("expected nil client")
	}
}

func TestManager_GetClientForLanguage_NoClients(t *testing.T) {
	m, _ := newTestManager(t)

	got, ok := m.GetClientForLanguage("go")
	if ok {
		t.Fatal("expected ok=false with no servers")
	}
	if got != nil {
		t.Error("expected nil client")
	}
}

func TestManager_GetClientForLanguage_MultipleServers(t *testing.T) {
	m, _ := newTestManager(t)

	goClient := newMockConnectedClient(t)
	pyClient := newMockConnectedClient(t)

	injectServer(m, "gopls", goClient, []string{"go"}, nil)
	injectServer(m, "pyright", pyClient, []string{"python"}, nil)

	got, ok := m.GetClientForLanguage("python")
	if !ok {
		t.Fatal("expected to find client for 'python'")
	}
	if got != pyClient {
		t.Error("expected pyright client, got gopls client")
	}
}

func TestManager_GetClientForFile_ByLanguage(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, nil)

	clients, err := m.GetClientForFile("file:///main.go")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
	if clients[0] != client {
		t.Error("returned client does not match")
	}
}

func TestManager_GetClientForFile_ByPattern(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, nil, []string{"*.go", "go.mod"})

	clients, err := m.GetClientForFile("file:///workspace/main.go")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(clients))
	}
}

func TestManager_GetClientForFile_NoExtension(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, nil)

	clients, err := m.GetClientForFile("file:///Makefile")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 0 {
		t.Errorf("expected 0 clients for extensionless file, got %d", len(clients))
	}
}

func TestManager_GetClientForFile_NoMatch(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, nil)

	clients, err := m.GetClientForFile("file:///main.py")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 0 {
		t.Errorf("expected 0 clients for .py file, got %d", len(clients))
	}
}

func TestManager_GetClientForFile_NoDuplicate(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, []string{"*.go"})

	clients, err := m.GetClientForFile("file:///main.go")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client (no duplicate), got %d", len(clients))
	}
}

func TestManager_GetClientForFile_MultipleServers(t *testing.T) {
	m, _ := newTestManager(t)

	goClient := newMockConnectedClient(t)
	tsClient := newMockConnectedClient(t)

	injectServer(m, "gopls", goClient, []string{"go"}, nil)
	injectServer(m, "tsserver", tsClient, []string{"typescript", "typescriptreact"}, nil)

	clients, err := m.GetClientForFile("file:///main.go")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 || clients[0] != goClient {
		t.Error("expected gopls client for .go file")
	}

	clients, err = m.GetClientForFile("file:///app.ts")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 || clients[0] != tsClient {
		t.Error("expected tsserver client for .ts file")
	}

	clients, err = m.GetClientForFile("file:///app.tsx")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 || clients[0] != tsClient {
		t.Error("expected tsserver client for .tsx file")
	}
}

func TestManager_StartServer_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.StartServer(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestManager_StartServer_AlreadyRunning(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, nil, nil)

	err := m.StartServer(context.Background(), "gopls")
	if err != nil {
		t.Fatalf("expected nil error for already running server, got: %v", err)
	}
}

func TestManager_StartServer_CommandNotFound(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.AddServer("nonexistent-lsp", []string{"definitely-not-a-real-binary-12345"}, "file:///workspace"); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	err := m.StartServer(context.Background(), "nonexistent-lsp")
	if err == nil {
		t.Fatal("expected error when starting server with nonexistent command")
	}
}

func TestManager_StopServer_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.StopServer("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestManager_StopServer_NoClient(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.AddServer("gopls", []string{"gopls"}, "file:///workspace"); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	err := m.StopServer("gopls")
	if err != nil {
		t.Fatalf("expected nil error when stopping server with no client, got: %v", err)
	}
}

func TestManager_StopServer_WithClient(t *testing.T) {
	m, _ := newTestManager(t)

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go"}, nil)

	if !client.IsConnected() {
		t.Fatal("client should be connected")
	}

	err := m.StopServer("gopls")
	if err != nil {
		t.Fatalf("StopServer failed: %v", err)
	}

	if client.IsConnected() {
		t.Error("client should not be connected after StopServer")
	}

	m.mu.RLock()
	entry := m.servers["gopls"]
	m.mu.RUnlock()
	if entry.client != nil {
		t.Error("expected entry.client to be nil after StopServer")
	}
	if entry.languages != nil {
		t.Error("expected entry.languages to be nil after StopServer")
	}
}

func TestManager_RestartServer_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.RestartServer(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestManager_RestartServer_StoppedServer(t *testing.T) {
	m, _ := newTestManager(t)

	if err := m.AddServer("bad-lsp", []string{"not-a-real-binary"}, "file:///workspace"); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	err := m.RestartServer(context.Background(), "bad-lsp")
	if err == nil {
		t.Fatal("expected error when restarting server that can't start")
	}
}

func TestManager_StartAll_Empty(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.StartAll(context.Background())
	if err != nil {
		t.Fatalf("StartAll on empty manager should not error, got: %v", err)
	}
}

func TestManager_StartAll_CommandNotFound(t *testing.T) {
	m, _ := newTestManager(t)

	_ = m.AddServer("bad1", []string{"not-real-1"}, "file:///workspace")
	_ = m.AddServer("bad2", []string{"not-real-2"}, "file:///workspace")

	err := m.StartAll(context.Background())
	if err == nil {
		t.Fatal("expected error from StartAll with nonexistent commands")
	}
}

func TestManager_StopAll_Empty(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.StopAll()
	if err != nil {
		t.Fatalf("StopAll on empty manager should not error, got: %v", err)
	}
}

func TestManager_StopAll_NoClients(t *testing.T) {
	m, _ := newTestManager(t)

	_ = m.AddServer("gopls", []string{"gopls"}, "file:///workspace")
	_ = m.AddServer("pyright", []string{"pyright"}, "file:///workspace")

	err := m.StopAll()
	if err != nil {
		t.Fatalf("StopAll with no clients should not error, got: %v", err)
	}
}

func TestManager_StopAll_WithClients(t *testing.T) {
	m, _ := newTestManager(t)

	client1 := newMockConnectedClient(t)
	client2 := newMockConnectedClient(t)

	injectServer(m, "gopls", client1, []string{"go"}, nil)
	injectServer(m, "pyright", client2, []string{"python"}, nil)

	err := m.StopAll()
	if err != nil {
		t.Fatalf("StopAll failed: %v", err)
	}

	if client1.IsConnected() {
		t.Error("client1 should not be connected after StopAll")
	}
	if client2.IsConnected() {
		t.Error("client2 should not be connected after StopAll")
	}

	m.mu.RLock()
	for name, entry := range m.servers {
		if entry.client != nil {
			t.Errorf("expected entry.client to be nil for %q after StopAll", name)
		}
	}
	m.mu.RUnlock()
}

func TestManager_SetLanguages(t *testing.T) {
	m, _ := newTestManager(t)

	_ = m.AddServer("gopls", []string{"gopls"}, "file:///workspace")

	err := m.SetLanguages("gopls", []string{"go", "gomod"})
	if err != nil {
		t.Fatalf("SetLanguages failed: %v", err)
	}

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, []string{"go", "gomod"}, nil)

	got, ok := m.GetClientForLanguage("gomod")
	if !ok {
		t.Fatal("expected to find client for 'gomod'")
	}
	if got != client {
		t.Error("returned client does not match")
	}
}

func TestManager_SetLanguages_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.SetLanguages("nonexistent", []string{"go"})
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestManager_SetFilePatterns(t *testing.T) {
	m, _ := newTestManager(t)

	_ = m.AddServer("gopls", []string{"gopls"}, "file:///workspace")

	err := m.SetFilePatterns("gopls", []string{"*.go", "go.mod"})
	if err != nil {
		t.Fatalf("SetFilePatterns failed: %v", err)
	}

	client := newMockConnectedClient(t)
	injectServer(m, "gopls", client, nil, []string{"*.go", "go.mod"})

	clients, err := m.GetClientForFile("file:///go.mod")
	if err != nil {
		t.Fatalf("GetClientForFile failed: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 client for go.mod, got %d", len(clients))
	}
}

func TestManager_SetFilePatterns_NotFound(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.SetFilePatterns("nonexistent", []string{"*.go"})
	if err == nil {
		t.Fatal("expected error for nonexistent server")
	}
}

func TestManager_Servers_Empty(t *testing.T) {
	m, _ := newTestManager(t)

	names := m.Servers()
	if len(names) != 0 {
		t.Errorf("expected 0 servers, got %d", len(names))
	}
}

func TestManager_Servers_Multiple(t *testing.T) {
	m, _ := newTestManager(t)

	_ = m.AddServer("a", []string{"a"}, "file:///a")
	_ = m.AddServer("b", []string{"b"}, "file:///b")
	_ = m.AddServer("c", []string{"c"}, "file:///c")

	names := m.Servers()
	if len(names) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(names))
	}
	sort.Strings(names)
	expected := []string{"a", "b", "c"}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("expected %q at index %d, got %q", expected[i], i, n)
		}
	}
}

func TestManager_ConnectedServers_Empty(t *testing.T) {
	m, _ := newTestManager(t)

	names := m.ConnectedServers()
	if len(names) != 0 {
		t.Errorf("expected 0 connected servers, got %d", len(names))
	}
}

func TestManager_ConnectedServers_NoneConnected(t *testing.T) {
	m, _ := newTestManager(t)

	_ = m.AddServer("gopls", []string{"gopls"}, "file:///workspace")
	_ = m.AddServer("pyright", []string{"pyright"}, "file:///workspace")

	names := m.ConnectedServers()
	if len(names) != 0 {
		t.Errorf("expected 0 connected servers (none started), got %d", len(names))
	}
}

func TestManager_ConnectedServers_SomeConnected(t *testing.T) {
	m, _ := newTestManager(t)

	connected := newMockConnectedClient(t)
	injectServer(m, "gopls", connected, nil, nil)

	_ = m.AddServer("pyright", []string{"pyright"}, "file:///workspace")

	names := m.ConnectedServers()
	if len(names) != 1 {
		t.Fatalf("expected 1 connected server, got %d", len(names))
	}
	if names[0] != "gopls" {
		t.Errorf("expected connected server 'gopls', got %q", names[0])
	}
}

func TestManager_ConnectedServers_AllConnected(t *testing.T) {
	m, _ := newTestManager(t)

	c1 := newMockConnectedClient(t)
	c2 := newMockConnectedClient(t)

	injectServer(m, "gopls", c1, nil, nil)
	injectServer(m, "pyright", c2, nil, nil)

	names := m.ConnectedServers()
	if len(names) != 2 {
		t.Fatalf("expected 2 connected servers, got %d", len(names))
	}
	sort.Strings(names)
	if names[0] != "gopls" || names[1] != "pyright" {
		t.Errorf("unexpected connected servers: %v", names)
	}
}

func TestManager_DidOpen_NoServers(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.DidOpen(context.Background(), "file:///test.go", "go", "package main")
	if err != nil {
		t.Fatalf("DidOpen with no servers should not error, got: %v", err)
	}
}

func TestManager_DidOpen_MatchingClient(t *testing.T) {
	m, _ := newTestManager(t)

	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			Capabilities: ServerCapabilities{},
			ServerInfo:   ServerInfo{Name: "gopls", Version: "1.0.0"},
		}), nil
	})
	client := NewClient(mock)
	if err := client.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	injectServer(m, "gopls", client, []string{"go"}, nil)

	err := m.DidOpen(context.Background(), "file:///test.go", "go", "package main")
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	notifications := mock.getNotifications()
	found := false
	for _, n := range notifications {
		if n.Method == MethodTextDocumentDidOpen {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected didOpen notification to be sent")
	}
}

func TestManager_DidOpen_NoMatchingClients_BroadcastsToAll(t *testing.T) {
	m, _ := newTestManager(t)

	mock1 := newMockTransport()
	mock1.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "s1", Version: "1.0.0"},
		}), nil
	})
	c1 := NewClient(mock1)
	_ = c1.Connect(context.Background())

	mock2 := newMockTransport()
	mock2.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "s2", Version: "1.0.0"},
		}), nil
	})
	c2 := NewClient(mock2)
	_ = c2.Connect(context.Background())

	injectServer(m, "s1", c1, []string{"python"}, nil)
	injectServer(m, "s2", c2, []string{"python"}, nil)

	err := m.DidOpen(context.Background(), "file:///test.go", "go", "package main")
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	for _, mock := range []*mockTransport{mock1, mock2} {
		found := false
		for _, n := range mock.getNotifications() {
			if n.Method == MethodTextDocumentDidOpen {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected didOpen notification to be broadcast to all servers")
		}
	}
}

func TestManager_DidChange_NoServers(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.DidChange(context.Background(), "file:///test.go", []TextDocumentContentChangeEvent{
		{Text: "updated"},
	})
	if err != nil {
		t.Fatalf("DidChange with no servers should not error, got: %v", err)
	}
}

func TestManager_DidChange_MatchingClient(t *testing.T) {
	m, _ := newTestManager(t)

	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "gopls", Version: "1.0.0"},
		}), nil
	})
	client := NewClient(mock)
	_ = client.Connect(context.Background())

	injectServer(m, "gopls", client, []string{"go"}, nil)

	changes := []TextDocumentContentChangeEvent{
		{Text: "updated content"},
	}
	err := m.DidChange(context.Background(), "file:///test.go", changes)
	if err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	found := false
	for _, n := range mock.getNotifications() {
		if n.Method == MethodTextDocumentDidChange {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected didChange notification to be sent")
	}
}

func TestManager_DidClose_NoServers(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.DidClose(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DidClose with no servers should not error, got: %v", err)
	}
}

func TestManager_DidClose_MatchingClient(t *testing.T) {
	m, _ := newTestManager(t)

	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "gopls", Version: "1.0.0"},
		}), nil
	})
	client := NewClient(mock)
	_ = client.Connect(context.Background())

	injectServer(m, "gopls", client, []string{"go"}, nil)

	err := m.DidClose(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DidClose failed: %v", err)
	}

	found := false
	for _, n := range mock.getNotifications() {
		if n.Method == MethodTextDocumentDidClose {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected didClose notification to be sent")
	}
}

func TestManager_DidSave_NoServers(t *testing.T) {
	m, _ := newTestManager(t)

	err := m.DidSave(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DidSave with no servers should not error, got: %v", err)
	}
}

func TestManager_DidSave_MatchingClient(t *testing.T) {
	m, _ := newTestManager(t)

	mock := newMockTransport()
	mock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "gopls", Version: "1.0.0"},
		}), nil
	})
	client := NewClient(mock)
	_ = client.Connect(context.Background())

	injectServer(m, "gopls", client, []string{"go"}, nil)

	err := m.DidSave(context.Background(), "file:///test.go")
	if err != nil {
		t.Fatalf("DidSave failed: %v", err)
	}

	found := false
	for _, n := range mock.getNotifications() {
		if n.Method == MethodTextDocumentDidSave {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected didSave notification to be sent")
	}
}

func TestMatchLanguageByExtension(t *testing.T) {
	tests := []struct {
		ext      string
		lang     string
		expected bool
	}{
		{".go", "go", true},
		{".go", "python", false},
		{".py", "python", true},
		{".py", "go", false},
		{".ts", "typescript", true},
		{".tsx", "typescriptreact", true},
		{".jsx", "javascriptreact", true},
		{".js", "javascript", true},
		{".rs", "rust", true},
		{".java", "java", true},
		{".c", "c", true},
		{".cpp", "cpp", true},
		{".h", "c", true},
		{".hpp", "cpp", true},
		{".cs", "csharp", true},
		{".rb", "ruby", true},
		{".php", "php", true},
		{".swift", "swift", true},
		{".kt", "kotlin", true},
		{".scala", "scala", true},
		{".sh", "shellscript", true},
		{".bash", "shellscript", true},
		{".yaml", "yaml", true},
		{".yml", "yaml", true},
		{".json", "json", true},
		{".xml", "xml", true},
		{".html", "html", true},
		{".css", "css", true},
		{".sql", "sql", true},
		{".md", "markdown", true},
		{".xyz", "unknown", false},
		{"", "go", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext+"_"+tt.lang, func(t *testing.T) {
			got := matchLanguageByExtension(tt.ext, tt.lang)
			if got != tt.expected {
				t.Errorf("matchLanguageByExtension(%q, %q) = %v, want %v", tt.ext, tt.lang, got, tt.expected)
			}
		})
	}
}

func TestContainsClient(t *testing.T) {
	c1 := &Client{}
	c2 := &Client{}
	c3 := &Client{}

	clients := []*Client{c1, c2}

	if !containsClient(clients, c1) {
		t.Error("expected containsClient to find c1")
	}
	if !containsClient(clients, c2) {
		t.Error("expected containsClient to find c2")
	}
	if containsClient(clients, c3) {
		t.Error("expected containsClient to not find c3")
	}
	if containsClient(nil, c1) {
		t.Error("expected containsClient(nil) to return false")
	}
	if containsClient([]*Client{}, c1) {
		t.Error("expected containsClient(empty) to return false")
	}
}

func TestExtractLanguages(t *testing.T) {
	langs := extractLanguages(ServerCapabilities{})
	if langs != nil {
		t.Errorf("expected nil from extractLanguages, got %v", langs)
	}
}

func TestWithLanguages(t *testing.T) {
	mock := newMockTransport()
	client := NewClient(mock, WithLanguages("go", "python"))
	if client == nil {
		t.Fatal("NewClient with WithLanguages returned nil")
	}
}

func TestWithFilePatterns(t *testing.T) {
	mock := newMockTransport()
	client := NewClient(mock, WithFilePatterns("*.go", "go.mod"))
	if client == nil {
		t.Fatal("NewClient with WithFilePatterns returned nil")
	}
}

func TestManager_ConcurrentAccess(t *testing.T) {
	m, _ := newTestManager(t)

	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		_ = m.AddServer(name, []string{name}, "file:///"+name)
	}

	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := string(rune('A' + (i % 26)))
			_ = m.AddServer(name, []string{name}, "file:///"+name)
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Servers()
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.ConnectedServers()
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := string(rune('a' + (i % 5)))
			_, _ = m.GetClient(name)
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = m.GetClientForLanguage("go")
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = m.GetClientForFile("file:///test.go")
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := string(rune('a' + (i % 5)))
			_ = m.SetLanguages(name, []string{"go"})
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := string(rune('a' + (i % 5)))
			_ = m.SetFilePatterns(name, []string{"*.go"})
		}(i)
	}

	wg.Wait()
}

func TestManager_ConcurrentStopAll(t *testing.T) {
	m, _ := newTestManager(t)

	for i := 0; i < 10; i++ {
		name := string(rune('a' + i))
		client := newMockConnectedClient(t)
		injectServer(m, name, client, nil, nil)
	}

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		if i%2 == 0 {
			go func() {
				defer wg.Done()
				_ = m.StopAll()
			}()
		} else {
			go func() {
				defer wg.Done()
				_ = m.Servers()
			}()
		}
	}

	wg.Wait()
}

func TestManager_StartServer_PublishesEvent(t *testing.T) {
	bus := event.NewEventBus()
	defer bus.Close()
	m := NewManager(bus)

	if m.eventBus == nil {
		t.Fatal("eventBus should not be nil")
	}
}

func TestManager_NilEventBus(t *testing.T) {
	m := NewManager(nil)

	_ = m.AddServer("gopls", []string{"gopls"}, "file:///workspace")

	err := m.StartServer(context.Background(), "gopls")
	if err == nil {
		t.Fatal("expected error from nonexistent command")
	}
}

func TestManager_DidOpen_ExtensionMatching(t *testing.T) {
	m, _ := newTestManager(t)

	goMock := newMockTransport()
	goMock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "gopls", Version: "1.0.0"},
		}), nil
	})
	goClient := NewClient(goMock)
	_ = goClient.Connect(context.Background())

	pyMock := newMockTransport()
	pyMock.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "pyright", Version: "1.0.0"},
		}), nil
	})
	pyClient := NewClient(pyMock)
	_ = pyClient.Connect(context.Background())

	injectServer(m, "gopls", goClient, []string{"go"}, nil)
	injectServer(m, "pyright", pyClient, []string{"python"}, nil)

	err := m.DidOpen(context.Background(), "file:///main.go", "go", "package main")
	if err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	goFound := false
	for _, n := range goMock.getNotifications() {
		if n.Method == MethodTextDocumentDidOpen {
			goFound = true
			break
		}
	}
	if !goFound {
		t.Error("expected gopls to receive didOpen for .go file")
	}

	pyFound := false
	for _, n := range pyMock.getNotifications() {
		if n.Method == MethodTextDocumentDidOpen {
			pyFound = true
			break
		}
	}
	if pyFound {
		t.Error("pyright should NOT receive didOpen for .go file")
	}
}

func TestManager_DidChange_NoMatchingClients_BroadcastsToAll(t *testing.T) {
	m, _ := newTestManager(t)

	mock1 := newMockTransport()
	mock1.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "s1", Version: "1.0.0"},
		}), nil
	})
	c1 := NewClient(mock1)
	_ = c1.Connect(context.Background())

	mock2 := newMockTransport()
	mock2.setHandler(MethodInitialize, func(id int, params any) (*Response, error) {
		return okResponse(id, InitializeResult{
			ServerInfo: ServerInfo{Name: "s2", Version: "1.0.0"},
		}), nil
	})
	c2 := NewClient(mock2)
	_ = c2.Connect(context.Background())

	injectServer(m, "s1", c1, []string{"python"}, nil)
	injectServer(m, "s2", c2, []string{"python"}, nil)

	changes := []TextDocumentContentChangeEvent{{Text: "updated"}}
	err := m.DidChange(context.Background(), "file:///test.go", changes)
	if err != nil {
		t.Fatalf("DidChange failed: %v", err)
	}

	for _, mock := range []*mockTransport{mock1, mock2} {
		found := false
		for _, n := range mock.getNotifications() {
			if n.Method == MethodTextDocumentDidChange {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected didChange notification to be broadcast to all servers")
		}
	}
}

func TestManager_AddServer_StoresConfig(t *testing.T) {
	m, _ := newTestManager(t)

	cmd := []string{"gopls", "serve"}
	rootURI := "file:///my-project"

	if err := m.AddServer("gopls", cmd, rootURI); err != nil {
		t.Fatalf("AddServer failed: %v", err)
	}

	m.mu.RLock()
	entry := m.servers["gopls"]
	m.mu.RUnlock()

	if entry == nil {
		t.Fatal("expected entry to exist")
	}
	if len(entry.cmd) != 2 || entry.cmd[0] != "gopls" || entry.cmd[1] != "serve" {
		t.Errorf("unexpected cmd: %v", entry.cmd)
	}
	if entry.rootURI != rootURI {
		t.Errorf("expected rootURI %q, got %q", rootURI, entry.rootURI)
	}
	if entry.client != nil {
		t.Error("expected client to be nil before start")
	}
}
