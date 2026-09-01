package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
)

type StdioTransport struct {
	cmd  *exec.Cmd
	in   io.WriteCloser
	dec  *FrameDecoder
	done chan struct{}
	mu   sync.Mutex
}

type StdioOptions struct {
	Command string
	Args    []string
	Env     map[string]string
	Dir     string
}

func NewStdioTransport(opts StdioOptions) (*StdioTransport, error) {
	if strings.TrimSpace(opts.Command) == "" {
		return nil, fmt.Errorf("mcp: stdio command is empty")
	}
	cmd := exec.Command(opts.Command, opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		keys := make([]string, 0, len(opts.Env))
		for k := range opts.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		env := os.Environ()
		base := make(map[string]string, len(env))
		for _, kv := range env {
			if i := strings.IndexByte(kv, '='); i > 0 {
				base[kv[:i]] = kv[i+1:]
			}
		}
		for _, k := range keys {
			base[k] = opts.Env[k]
		}
		final := make([]string, 0, len(base))
		for k, v := range base {
			final = append(final, k+"="+v)
		}
		sort.Strings(final)
		cmd.Env = final
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdio stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdio stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp: start stdio process: %w", err)
	}

	return &StdioTransport{
		cmd:  cmd,
		in:   stdin,
		dec:  NewFrameDecoder(stdout),
		done: make(chan struct{}),
	}, nil
}

func (t *StdioTransport) Connect(_ context.Context) error {
	return nil
}

func (t *StdioTransport) Write(data []byte) error {
	select {
	case <-t.done:
		return ErrClosed
	default:
	}
	_, err := t.in.Write(append(append([]byte(nil), data...), '\n'))
	if err != nil {
		return fmt.Errorf("mcp: stdio write: %w", err)
	}
	return nil
}

func (t *StdioTransport) ReadFrame() ([]byte, error) {
	select {
	case <-t.done:
		return nil, ErrClosed
	default:
	}
	frame, err := t.dec.Decode()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrClosed
		}
		return nil, err
	}
	if len(frame) == 0 {
		return nil, nil
	}
	return frame, nil
}

func (t *StdioTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	select {
	case <-t.done:
		return nil
	default:
	}

	_ = t.in.Close()
	if t.cmd.Process != nil {
		_ = t.cmd.Process.Signal(syscall.SIGTERM)
	}
	close(t.done)
	go func() {
		_ = t.cmd.Wait()
	}()
	return nil
}
