package lsp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type Transport interface {
	Write(data []byte) error
	ReadFrame() ([]byte, error)
	Close() error
}

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
		return nil, fmt.Errorf("lsp: stdio command is empty")
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
		return nil, fmt.Errorf("lsp: stdio stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp: stdio stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("lsp: start stdio process: %w", err)
	}

	return &StdioTransport{
		cmd:  cmd,
		in:   stdin,
		dec:  NewFrameDecoder(stdout),
		done: make(chan struct{}),
	}, nil
}

func (t *StdioTransport) Write(data []byte) error {
	select {
	case <-t.done:
		return ErrClosed
	default:
	}
	header := "Content-Length: " + strconv.Itoa(len(data)) + "\r\n\r\n"
	_, err := t.in.Write(append(append([]byte(nil), header...), data...))
	if err != nil {
		return fmt.Errorf("lsp: stdio write: %w", err)
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

type FrameDecoder struct {
	br     *bufio.Reader
	toRead int
}

func NewFrameDecoder(r io.Reader) *FrameDecoder {
	return &FrameDecoder{br: bufio.NewReader(r)}
}

func (d *FrameDecoder) Decode() ([]byte, error) {
	for {
		if d.toRead > 0 {
			return d.readBody()
		}
		line, err := d.br.ReadBytes('\n')
		if err != nil && len(line) == 0 {
			return nil, err
		}
		trimmed := bytes.TrimSpace(bytes.TrimRight(line, "\r\n"))
		if strings.HasPrefix(string(trimmed), "Content-Length:") {
			raw, perr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(string(trimmed), "Content-Length:")))
			if perr != nil || raw <= 0 {
				return nil, fmt.Errorf("%w: invalid Content-Length", ErrProtocol)
			}
			d.toRead = raw
			continue
		}
		if len(trimmed) == 0 {
			continue
		}
		return trimmed, nil
	}
}

func (d *FrameDecoder) readBody() ([]byte, error) {
	peek, err := d.br.Peek(2)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if len(peek) >= 1 && (peek[0] == '\r' || peek[0] == '\n') {
		_, _ = d.br.ReadByte()
		if len(peek) >= 2 && (peek[1] == '\r' || peek[1] == '\n') {
			_, _ = d.br.ReadByte()
		}
	}

	body := make([]byte, d.toRead)
	if _, err := io.ReadFull(d.br, body); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	d.toRead = 0
	return body, nil
}
