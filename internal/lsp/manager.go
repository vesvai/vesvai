package lsp

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/vesvai/vesvai/internal/event"
)

type Manager struct {
	mu       sync.RWMutex
	servers  map[string]*serverEntry
	eventBus event.EventBus
}

type serverEntry struct {
	client    *Client
	cmd       []string
	rootURI   string
	opts      []ClientOption
	languages []string
	patterns  []string
}

func NewManager(eventBus event.EventBus) *Manager {
	return &Manager{
		servers:  make(map[string]*serverEntry),
		eventBus: eventBus,
	}
}

func (m *Manager) AddServer(name string, cmd []string, rootURI string, opts ...ClientOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[name]; exists {
		return fmt.Errorf("server %q already exists", name)
	}

	m.servers[name] = &serverEntry{
		cmd:     cmd,
		rootURI: rootURI,
		opts:    opts,
	}

	return nil
}

func (m *Manager) RemoveServer(name string) error {
	m.mu.Lock()
	entry, exists := m.servers[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("server %q not found", name)
	}
	delete(m.servers, name)
	m.mu.Unlock()

	if entry.client != nil {
		return entry.client.Close()
	}
	return nil
}

func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, exists := m.servers[name]
	if !exists || entry.client == nil {
		return nil, false
	}
	return entry.client, true
}

func (m *Manager) GetClientForLanguage(language string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, entry := range m.servers {
		if entry.client == nil {
			continue
		}
		if slices.Contains(entry.languages, language) {
			return entry.client, true
		}
	}
	return nil, false
}

func (m *Manager) GetClientForFile(uri string) ([]*Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(uri))
	if ext == "" {
		return nil, nil
	}

	var clients []*Client
	for _, entry := range m.servers {
		if entry.client == nil {
			continue
		}

		for _, lang := range entry.languages {
			if matchLanguageByExtension(ext, lang) {
				clients = append(clients, entry.client)
				break
			}
		}

		for _, pattern := range entry.patterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(uri)); matched {
				if !containsClient(clients, entry.client) {
					clients = append(clients, entry.client)
				}
				break
			}
		}
	}

	return clients, nil
}

func (m *Manager) StartServer(ctx context.Context, name string) error {
	m.mu.RLock()
	entry, exists := m.servers[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %q not found", name)
	}

	if entry.client != nil && entry.client.IsConnected() {
		return nil
	}

	transport := NewStdioTransport(entry.cmd[0], entry.cmd[1:]...)
	client := NewClient(transport, entry.opts...)

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to start server %q: %w", name, err)
	}

	caps := client.Capabilities()
	entry.languages = extractLanguages(caps)
	entry.client = client

	if m.eventBus != nil {
		m.eventBus.Publish(ctx, NewLSPEvent(EventLSPServerStart, ServerStartEventData{
			ServerName: name,
			ServerInfo: client.ServerInfo(),
			Languages:  entry.languages,
		}))
	}

	return nil
}

func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	names := slices.Collect(maps.Keys(m.servers))
	m.mu.RUnlock()

	var firstErr error
	for _, name := range names {
		if err := m.StartServer(ctx, name); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) StopServer(name string) error {
	m.mu.RLock()
	entry, exists := m.servers[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %q not found", name)
	}

	if entry.client == nil {
		return nil
	}

	err := entry.client.Close()
	entry.client = nil
	entry.languages = nil

	return err
}

func (m *Manager) StopAll() error {
	m.mu.RLock()
	entries := maps.Clone(m.servers)
	m.mu.RUnlock()

	var firstErr error
	for _, entry := range entries {
		if entry.client != nil {
			if err := entry.client.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
			m.mu.Lock()
			entry.client = nil
			entry.languages = nil
			m.mu.Unlock()
		}
	}
	return firstErr
}

func (m *Manager) RestartServer(ctx context.Context, name string) error {
	if err := m.StopServer(name); err != nil {
		m.mu.RLock()
		_, exists := m.servers[name]
		m.mu.RUnlock()
		if !exists {
			return err
		}
	}
	return m.StartServer(ctx, name)
}

func (m *Manager) DidOpen(ctx context.Context, uri, languageId, text string) error {
	clients, err := m.GetClientForFile(uri)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		clients = m.allConnectedClients()
	}

	var firstErr error
	for _, client := range clients {
		if err := client.DidOpen(ctx, uri, languageId, text); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) DidChange(ctx context.Context, uri string, changes []TextDocumentContentChangeEvent) error {
	clients, err := m.GetClientForFile(uri)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		clients = m.allConnectedClients()
	}

	var firstErr error
	for _, client := range clients {
		if err := client.DidChange(ctx, uri, changes); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) DidClose(ctx context.Context, uri string) error {
	clients, err := m.GetClientForFile(uri)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		clients = m.allConnectedClients()
	}

	var firstErr error
	for _, client := range clients {
		if err := client.DidClose(ctx, uri); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (m *Manager) DidSave(ctx context.Context, uri string) error {
	clients, err := m.GetClientForFile(uri)
	if err != nil {
		return err
	}

	if len(clients) == 0 {
		clients = m.allConnectedClients()
	}

	var firstErr error
	for _, client := range clients {
		if err := client.DidSave(ctx, uri); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func WithLanguages(languages ...string) ClientOption {
	return func(c *Client) {
	}
}

func WithFilePatterns(patterns ...string) ClientOption {
	return func(c *Client) {
	}
}

func (m *Manager) SetLanguages(name string, languages []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server %q not found", name)
	}
	entry.languages = languages
	return nil
}

func (m *Manager) SetFilePatterns(name string, patterns []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, exists := m.servers[name]
	if !exists {
		return fmt.Errorf("server %q not found", name)
	}
	entry.patterns = patterns
	return nil
}

func (m *Manager) Servers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return slices.Collect(maps.Keys(m.servers))
}

func (m *Manager) ConnectedServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name, entry := range m.servers {
		if entry.client != nil && entry.client.IsConnected() {
			names = append(names, name)
		}
	}
	return names
}

func (m *Manager) allConnectedClients() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var clients []*Client
	for _, entry := range m.servers {
		if entry.client != nil && entry.client.IsConnected() {
			clients = append(clients, entry.client)
		}
	}
	return clients
}

func extractLanguages(_ ServerCapabilities) []string {
	return nil
}

func matchLanguageByExtension(ext, languageID string) bool {
	extToLang := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".tsx":   "typescriptreact",
		".jsx":   "javascriptreact",
		".rs":    "rust",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".sh":    "shellscript",
		".bash":  "shellscript",
		".yaml":  "yaml",
		".yml":   "yaml",
		".json":  "json",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".sql":   "sql",
		".md":    "markdown",
	}

	if lang, ok := extToLang[ext]; ok {
		return lang == languageID
	}
	return false
}

func containsClient(clients []*Client, target *Client) bool {
	return slices.Contains(clients, target)
}
