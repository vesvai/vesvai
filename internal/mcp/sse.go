package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type SSETransport struct {
	baseURL string
	headers map[string]string

	mu           sync.Mutex
	client       *http.Client
	pending      map[RequestID]chan *JSONRPCResponse
	notifHandler func(JSONRPCNotification)
	nextID       int64
	closed       bool
	eventSource  io.ReadCloser
	cancel       context.CancelFunc
	done         chan struct{}
}

func NewSSETransport(baseURL string, headers map[string]string) *SSETransport {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &SSETransport{
		baseURL: strings.TrimRight(baseURL, "/"),
		headers: headers,
		client:  &http.Client{Timeout: 30 * time.Second},
		pending: make(map[RequestID]chan *JSONRPCResponse),
		done:    make(chan struct{}),
	}
}

func (t *SSETransport) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	sseURL := t.baseURL
	if !strings.Contains(sseURL, "/sse") {
		sseURL = sseURL + "/sse"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", sseURL, nil)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Connection", "keep-alive")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		cancel()
		return fmt.Errorf("failed to connect to SSE endpoint: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		cancel()
		return fmt.Errorf("SSE connection failed with status: %d", resp.StatusCode)
	}

	t.eventSource = resp.Body
	go t.readSSELoop(ctx)
	go t.waitForContext(ctx)

	return nil
}

func (t *SSETransport) waitForContext(ctx context.Context) {
	<-ctx.Done()
	t.Close()
}

func (t *SSETransport) readSSELoop(ctx context.Context) {
	defer close(t.done)

	scanner := bufio.NewScanner(t.eventSource)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var eventData strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			data := eventData.String()
			if data != "" && eventType == "message" {
				t.handleSSEMessage(data)
			}
			eventType = ""
			eventData.Reset()
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			if eventData.Len() > 0 {
				eventData.WriteString("\n")
			}
			eventData.WriteString(strings.TrimPrefix(line, "data:"))
		}
	}
}

func (t *SSETransport) handleSSEMessage(data string) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return
	}

	if _, hasID := raw["id"]; hasID {
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(data), &resp); err == nil {
			t.mu.Lock()
			ch, ok := t.pending[resp.ID]
			if ok {
				delete(t.pending, resp.ID)
			}
			t.mu.Unlock()

			if ok {
				ch <- &resp
			}
			return
		}
	}

	var notif JSONRPCNotification
	if err := json.Unmarshal([]byte(data), &notif); err == nil && notif.Method != "" {
		t.mu.Lock()
		handler := t.notifHandler
		t.mu.Unlock()

		if handler != nil {
			handler(notif)
		}
	}
}

func (t *SSETransport) SendRequest(ctx context.Context, method string, params any) (*JSONRPCResponse, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}

	t.nextID++
	id := IntID(t.nextID)

	ch := make(chan *JSONRPCResponse, 1)
	t.pending[id] = ch
	t.mu.Unlock()

	req := JSONRPCRequest{
		JSONRPC: jsonrpcVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	messageURL := t.baseURL
	if !strings.Contains(messageURL, "/message") {
		messageURL = messageURL + "/message"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", messageURL, strings.NewReader(string(data)))
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	select {
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	case rpcResp := <-ch:
		return rpcResp, nil
	case <-t.done:
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("transport closed")
	}
}

func (t *SSETransport) SendNotification(ctx context.Context, method string, params any) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport is closed")
	}
	t.mu.Unlock()

	notif := JSONRPCNotification{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	messageURL := t.baseURL
	if !strings.Contains(messageURL, "/message") {
		messageURL = messageURL + "/message"
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", messageURL, strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (t *SSETransport) SetNotificationHandler(handler func(notification JSONRPCNotification)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.notifHandler = handler
}

func (t *SSETransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true

	for id, ch := range t.pending {
		close(ch)
		delete(t.pending, id)
	}

	if t.cancel != nil {
		t.cancel()
	}

	t.mu.Unlock()

	if t.eventSource != nil {
		t.eventSource.Close()
	}

	return nil
}

func SSETransportFromURL(rawURL string, headers map[string]string) (*SSETransport, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("URL must use http or https scheme")
	}

	return NewSSETransport(u.String(), headers), nil
}
