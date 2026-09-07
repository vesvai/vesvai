package acp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPTransportInitialize(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	body := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":1,"clientInfo":{"name":"test","version":"1.0.0"}}}`
	req := httptest.NewRequest("POST", "/acp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Acp-Connection-Id") == "" {
		t.Error("expected Acp-Connection-Id header")
	}
	var resp struct {
		Result struct {
			ProtocolVersion int `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Result.ProtocolVersion != 1 {
		t.Errorf("expected protocol version 1, got %d", resp.Result.ProtocolVersion)
	}
}

func TestHTTPTransportMissingContentType(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("POST", "/acp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", w.Code)
	}
}

func TestHTTPTransportUnknownConnection(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	body := `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/tmp"}}`
	req := httptest.NewRequest("POST", "/acp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Acp-Connection-Id", "unknown")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHTTPTransportMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("PUT", "/acp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestHTTPTransportGetStreamMissingAccept(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("GET", "/acp", nil)
	req.Header.Set("Acp-Connection-Id", "test")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotAcceptable {
		t.Errorf("expected 406, got %d", w.Code)
	}
}

func TestHTTPTransportGetStreamMissingConnID(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("GET", "/acp", nil)
	req.Header.Set("Accept", "text/event-stream")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHTTPTransportDelete(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("DELETE", "/acp", nil)
	req.Header.Set("Acp-Connection-Id", "test")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

func TestHTTPTransportDeleteMissingConnID(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("DELETE", "/acp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHTTPTransportFullFlow(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	body := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":1}}`
	req := httptest.NewRequest("POST", "/acp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("initialize failed: %d", w.Code)
	}
	connID := w.Header().Get("Acp-Connection-Id")
	if connID == "" {
		t.Fatal("missing connection id")
	}

	done := make(chan struct{})
	go func() {
		req := httptest.NewRequest("GET", "/acp", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Acp-Connection-Id", connID)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)

	body = `{"jsonrpc":"2.0","id":1,"method":"session/new","params":{"cwd":"/tmp"}}`
	req = httptest.NewRequest("POST", "/acp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Acp-Connection-Id", connID)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", w.Code)
	}
}

func TestHTTPTransportGetStreamUnknownConn(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("GET", "/acp", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Acp-Connection-Id", "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestStdioTransport(t *testing.T) {
	tr := NewStdioTransport()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgCh, err := tr.Start(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cancel()
	_, ok := <-msgCh
	if ok {
		t.Error("expected channel to close after cancel")
	}
}

func TestStdioTransportSend(t *testing.T) {
	tr := NewStdioTransport()
	err := tr.Send(Message{Body: json.RawMessage(`{"test":true}`)})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
}

func TestStdioTransportClose(t *testing.T) {
	tr := NewStdioTransport()
	if err := tr.Close(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHTTPTransportPostNoContentType(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	req := httptest.NewRequest("POST", "/acp", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnsupportedMediaType {
		t.Errorf("expected 415, got %d", w.Code)
	}
}

func TestHTTPTransportBadMethod(t *testing.T) {
	srv := newTestServer(t)
	handler := srv.HTTPHandler()

	for _, method := range []string{"PATCH", "OPTIONS", "HEAD"} {
		req := httptest.NewRequest(method, "/acp", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected 405, got %d", method, w.Code)
		}
	}
}

func TestExtractSessionIDFromParams(t *testing.T) {
	if got := extractSessionIDFromParams(json.RawMessage(`{"sessionId":"sess_1"}`)); got != "sess_1" {
		t.Errorf("expected sess_1, got %q", got)
	}
	if got := extractSessionIDFromParams(json.RawMessage(`{}`)); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	if got := extractSessionIDFromParams(nil); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	if got := extractSessionIDFromParams(json.RawMessage(`invalid`)); got != "" {
		t.Errorf("expected empty for invalid json, got %q", got)
	}
}
