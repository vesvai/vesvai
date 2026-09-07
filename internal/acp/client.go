package acp

import (
	"context"
	"fmt"
	"sync"
	"time"

	json "github.com/goccy/go-json"
	"github.com/google/uuid"
)

type Client struct {
	send    func(json.RawMessage) error
	pending map[string]chan json.RawMessage
	mu      sync.Mutex
	timeout time.Duration
}

func NewClient(sendFn func(json.RawMessage) error) *Client {
	return &Client{
		send:    sendFn,
		pending: make(map[string]chan json.RawMessage),
		timeout: 60 * time.Second,
	}
}

func (c *Client) DeliverResponse(id any, body json.RawMessage) {
	idStr := jsonIDToString(id)
	if idStr == "" {
		return
	}
	c.mu.Lock()
	ch, ok := c.pending[idStr]
	delete(c.pending, idStr)
	c.mu.Unlock()
	if ok {
		select {
		case ch <- body:
		default:
		}
	}
}

func (c *Client) call(ctx context.Context, method string, params any, timeout time.Duration) (json.RawMessage, error) {
	id := uuid.NewString()
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("acp: marshal client request: %w", err)
	}

	ch := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := c.send(body); err != nil {
		return nil, fmt.Errorf("acp: send client request: %w", err)
	}

	if timeout <= 0 {
		timeout = c.timeout
	}
	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("acp: timeout waiting for client response")
	}
}

func (c *Client) ReadTextFile(ctx context.Context, sessionID, path string, offset, limit int) (string, error) {
	params := map[string]any{
		"sessionId": sessionID,
		"path":      path,
	}
	if offset > 0 {
		params["line"] = offset
	}
	if limit > 0 {
		params["limit"] = limit
	}
	resp, err := c.call(ctx, "fs/read_text_file", params, c.timeout)
	if err != nil {
		return "", err
	}
	return extractResultContent(resp), nil
}

func (c *Client) WriteTextFile(ctx context.Context, sessionID, path, content string) error {
	params := map[string]any{
		"sessionId": sessionID,
		"path":      path,
		"content":   content,
	}
	_, err := c.call(ctx, "fs/write_text_file", params, c.timeout)
	return err
}

func (c *Client) RequestPermission(ctx context.Context, sessionID, toolCallID, title, kind string, options []PermissionOption) (string, error) {
	toolCall := map[string]any{"toolCallId": toolCallID}
	if title != "" {
		toolCall["title"] = title
	}
	if kind != "" {
		toolCall["kind"] = kind
	}
	params := map[string]any{
		"sessionId": sessionID,
		"toolCall":  toolCall,
		"options":   options,
	}
	resp, err := c.call(ctx, "session/request_permission", params, c.timeout)
	if err != nil {
		return "", err
	}
	return extractResultOutcome(resp), nil
}

func jsonIDToString(id any) string {
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case json.Number:
		return v.String()
	}
	return ""
}

func extractResultContent(resp json.RawMessage) string {
	var r struct {
		Result struct {
			Content string `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return ""
	}
	return r.Result.Content
}

func extractResultOutcome(resp json.RawMessage) string {
	var r struct {
		Result struct {
			Outcome string `json:"outcome"`
		} `json:"result"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return ""
	}
	return r.Result.Outcome
}
