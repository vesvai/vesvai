package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type Transport interface {
	Start(ctx context.Context) error
	SendRequest(ctx context.Context, method string, params any) (*JSONRPCResponse, error)
	SendNotification(ctx context.Context, method string, params any) error
	SetNotificationHandler(handler func(notification JSONRPCNotification))
	Close() error
}

type StdioTransport struct {
	command string
	args    []string
	env     []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu           sync.Mutex
	pending      map[RequestID]chan *JSONRPCResponse
	notifHandler func(JSONRPCNotification)
	nextID       int64
	closed       bool
	done         chan struct{}
	cancel       context.CancelFunc
}

func NewStdioTransport(command string, args ...string) *StdioTransport {
	return &StdioTransport{
		command: command,
		args:    args,
		pending: make(map[RequestID]chan *JSONRPCResponse),
		done:    make(chan struct{}),
	}
}

func (t *StdioTransport) SetEnv(env []string) {
	t.env = env
}

func (t *StdioTransport) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel

	t.cmd = exec.CommandContext(ctx, t.command, t.args...)
	t.cmd.Env = t.env

	var err error
	t.stdin, err = t.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	t.stdout, err = t.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	t.stderr, err = t.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := t.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	go t.readLoop()
	go t.waitProcess()

	return nil
}

func (t *StdioTransport) waitProcess() {
	if t.cmd != nil {
		t.cmd.Wait()
	}
	close(t.done)
}

func (t *StdioTransport) readLoop() {
	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			continue
		}

		if _, hasID := raw["id"]; hasID {
			var resp JSONRPCResponse
			if err := json.Unmarshal(line, &resp); err == nil {
				t.mu.Lock()
				ch, ok := t.pending[resp.ID]
				if ok {
					delete(t.pending, resp.ID)
				}
				t.mu.Unlock()

				if ok {
					ch <- &resp
				}
				continue
			}
		}

		var notif JSONRPCNotification
		if err := json.Unmarshal(line, &notif); err == nil && notif.Method != "" {
			t.mu.Lock()
			handler := t.notifHandler
			t.mu.Unlock()

			if handler != nil {
				handler(notif)
			}
		}
	}
}

func (t *StdioTransport) SendRequest(ctx context.Context, method string, params any) (*JSONRPCResponse, error) {
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

	data = append(data, '\n')

	t.mu.Lock()
	stdin := t.stdin
	t.mu.Unlock()

	if stdin == nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("stdin not available")
	}

	if _, err := stdin.Write(data); err != nil {
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	select {
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	case <-t.done:
		t.mu.Lock()
		delete(t.pending, id)
		t.mu.Unlock()
		return nil, fmt.Errorf("transport closed")
	}
}

func (t *StdioTransport) SendNotification(ctx context.Context, method string, params any) error {
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

	data = append(data, '\n')

	t.mu.Lock()
	stdin := t.stdin
	t.mu.Unlock()

	if stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write notification: %w", err)
	}

	return nil
}

func (t *StdioTransport) SetNotificationHandler(handler func(notification JSONRPCNotification)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.notifHandler = handler
}

func (t *StdioTransport) Close() error {
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

	if t.stdin != nil {
		t.stdin.Close()
	}

	return nil
}
