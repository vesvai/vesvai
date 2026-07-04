package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type Transport interface {
	Start(ctx context.Context) error

	SendRequest(ctx context.Context, id int, method string, params any) (*Response, error)

	SendNotification(ctx context.Context, method string, params any) error

	ReadMessage(ctx context.Context) (any, error)

	SetNotificationHandler(handler func(notification Notification))

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
	pending      map[RequestID]chan *Response
	notifHandler func(Notification)
	nextID       int
	closed       bool
	done         chan struct{}
	cancel       context.CancelFunc
}

func NewStdioTransport(command string, args ...string) *StdioTransport {
	return &StdioTransport{
		command: command,
		args:    args,
		pending: make(map[RequestID]chan *Response),
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
		return fmt.Errorf("failed to start LSP server: %w", err)
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
	reader := bufio.NewReader(t.stdout)

	for {
		contentLength := -1
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length: ") {
				val := strings.TrimPrefix(line, "Content-Length: ")
				val = strings.TrimSpace(val)
				contentLength, err = strconv.Atoi(val)
				if err != nil {
					continue
				}
			}
		}

		if contentLength < 0 {
			continue
		}

		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			return
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			continue
		}

		if _, hasID := raw["id"]; hasID {
			var resp Response
			if err := json.Unmarshal(body, &resp); err == nil {
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

		var notif Notification
		if err := json.Unmarshal(body, &notif); err == nil && notif.Method != "" {
			t.mu.Lock()
			handler := t.notifHandler
			t.mu.Unlock()

			if handler != nil {
				handler(notif)
			}
		}
	}
}

func (t *StdioTransport) SendRequest(ctx context.Context, id int, method string, params any) (*Response, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport is closed")
	}

	reqID := IntID(int64(id))
	ch := make(chan *Response, 1)
	t.pending[reqID] = ch
	t.mu.Unlock()

	req := Request{
		JSONRPC: jsonrpcVersion,
		ID:      reqID,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, reqID)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := t.writeMessage(data); err != nil {
		t.mu.Lock()
		delete(t.pending, reqID)
		t.mu.Unlock()
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	select {
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, reqID)
		t.mu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	case <-t.done:
		t.mu.Lock()
		delete(t.pending, reqID)
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

	notif := Notification{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	return t.writeMessage(data)
}

func (t *StdioTransport) ReadMessage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("not implemented: use SetNotificationHandler for server-initiated messages")
}

func (t *StdioTransport) SetNotificationHandler(handler func(notification Notification)) {
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

func (t *StdioTransport) writeMessage(data []byte) error {
	t.mu.Lock()
	stdin := t.stdin
	t.mu.Unlock()

	if stdin == nil {
		return fmt.Errorf("stdin not available")
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}
