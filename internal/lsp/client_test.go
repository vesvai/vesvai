package lsp

import (
	"context"
	json "github.com/goccy/go-json"
	"sync"
	"testing"
	"time"
)

type scriptedTransport struct {
	frames chan []byte
	closed chan struct{}
}

func newScriptedTransport() *scriptedTransport {
	return &scriptedTransport{
		frames: make(chan []byte, 16),
		closed: make(chan struct{}),
	}
}

func (t *scriptedTransport) Write(data []byte) error {
	select {
	case <-t.closed:
		return ErrClosed
	default:
	}
	var req struct {
		ID     int             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(data, &req)
	switch req.Method {
	case MethodInitialize:
		t.frames <- scriptedResponse(req.ID, map[string]any{
			"capabilities": map[string]any{},
			"serverInfo":   map[string]any{"name": "fake", "version": "1.0"},
		})
	case MethodShutdown:
		t.frames <- scriptedResponse(req.ID, map[string]any{})
	case MethodDidOpen:
		var p struct {
			TextDocument map[string]any `json:"textDocument"`
		}
		_ = json.Unmarshal(req.Params, &p)
		uri := ""
		if td := p.TextDocument; td != nil {
			if v, ok := td["uri"]; ok {
				if s, ok := v.(string); ok {
					uri = s
				}
			}
		}
		t.frames <- scriptedNotification(MethodPublishDiagnostics, map[string]any{
			"uri":     uri,
			"version": 1,
			"diagnostics": []map[string]any{
				{
					"severity": 1,
					"message":  "fake error",
					"range": map[string]any{
						"start": map[string]any{"line": 0, "character": 0},
						"end":   map[string]any{"line": 0, "character": 1},
					},
				},
			},
		})
	case MethodExit:
		_ = t.Close()
	}
	return nil
}

func (t *scriptedTransport) ReadFrame() ([]byte, error) {
	select {
	case <-t.closed:
		return nil, ErrClosed
	case f := <-t.frames:
		return f, nil
	}
}

func (t *scriptedTransport) Close() error {
	select {
	case <-t.closed:
		return nil
	default:
	}
	close(t.closed)
	return nil
}

func scriptedResponse(id int, result map[string]any) []byte {
	data, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
	return data
}

func scriptedNotification(method string, params any) []byte {
	data, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	})
	return data
}

type diagCollector struct {
	mu     sync.Mutex
	uri    string
	diags  []Diagnostic
	notify chan struct{}
}

func newDiagCollector() *diagCollector {
	return &diagCollector{notify: make(chan struct{}, 4)}
}

func (d *diagCollector) onDiag(uri string, _ int, diags []Diagnostic) {
	d.mu.Lock()
	d.uri = uri
	d.diags = diags
	d.mu.Unlock()
	select {
	case d.notify <- struct{}{}:
	default:
	}
}

func (d *diagCollector) wait(t *testing.T, timeout time.Duration) {
	deadline := time.After(timeout)
	select {
	case <-d.notify:
		return
	case <-deadline:
		t.Fatal("timed out waiting for diagnostics")
	}
}

func TestClientInitializeAndDiagnostics(t *testing.T) {
	tr := newScriptedTransport()
	collector := newDiagCollector()
	c := NewClient(tr, ClientOptions{Name: "fake", Timeout: 2 * time.Second, OnDiag: collector.onDiag})
	defer c.Close()

	info, err := c.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.ServerInfo.Name != "fake" {
		t.Fatalf("info = %+v", info)
	}

	if err := c.DidOpen(context.Background(), TextDocumentItem{
		URI:        "file:///tmp/main.go",
		LanguageID: "go",
		Version:    1,
		Text:       "package main\n",
	}); err != nil {
		t.Fatal(err)
	}

	collector.wait(t, 3*time.Second)
	collector.mu.Lock()
	defer collector.mu.Unlock()
	if collector.uri != "file:///tmp/main.go" {
		t.Fatalf("uri = %q", collector.uri)
	}
	if len(collector.diags) != 1 || collector.diags[0].Message != "fake error" {
		t.Fatalf("diags = %+v", collector.diags)
	}
	if collector.diags[0].Severity != 1 || collector.diags[0].Line() != 0 {
		t.Fatalf("diag = %+v", collector.diags[0])
	}
}

func TestClientShutdown(t *testing.T) {
	tr := newScriptedTransport()
	collector := newDiagCollector()
	c := NewClient(tr, ClientOptions{Name: "fake", Timeout: 2 * time.Second, OnDiag: collector.onDiag})
	defer c.Close()

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := c.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
