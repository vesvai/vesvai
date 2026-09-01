package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type SSETransport struct {
	url      string
	headers  map[string]string
	client   *http.Client
	session  string
	endpoint string

	resp      *http.Response
	postMu    sync.Mutex
	postWait  chan struct{}
	postResp  chan []byte
	postError chan error

	readCh chan []byte
	errCh  chan error

	endpointOnce sync.Once
	closeOnce    sync.Once
}

type SSEOptions struct {
	URL     string
	Headers map[string]string
	Client  *http.Client
}

func NewSSETransport(opts SSEOptions) (*SSETransport, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("mcp: sse url is empty")
	}
	if opts.Client == nil {
		opts.Client = &http.Client{}
	}
	return &SSETransport{
		url:      opts.URL,
		headers:  opts.Headers,
		client:   opts.Client,
		postWait: make(chan struct{}),
		readCh:   make(chan []byte, 16),
		errCh:    make(chan error, 1),
	}, nil
}

func (t *SSETransport) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url, nil)
	if err != nil {
		return fmt.Errorf("mcp: sse build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("mcp: sse open stream: %w", err)
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
		return fmt.Errorf("mcp: sse endpoint %s: HTTP %d: %s", t.url, resp.StatusCode, string(body))
	}

	t.resp = resp
	go t.readLoop(resp.Body)

	select {
	case <-ctx.Done():
		t.Close()
		return fmt.Errorf("mcp: sse connect: %w", ctx.Err())
	case err := <-t.errCh:
		t.Close()
		return fmt.Errorf("mcp: sse connect: %w", err)
	case <-t.postWait:
		return nil
	}
}

func (t *SSETransport) readLoop(body io.ReadCloser) {
	defer body.Close()
	defer t.fail(ErrClosed)

	rd := newSSELineReader(body)
	event := ""
	var data []string

	emit := func() {
		switch event {
		case "endpoint":
			if len(data) > 0 {
				t.endpoint = strings.Join(data, "\n")
				t.endpointOnce.Do(func() { close(t.postWait) })
			}
		case "message":
			if len(data) > 0 {
				t.readCh <- []byte(strings.Join(data, "\n"))
			}
		}
		event = ""
		data = nil
	}

	for {
		line, err := rd.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = ErrClosed
			}
			t.fail(err)
			return
		}
		if len(line) == 0 {
			emit()
			continue
		}
		if line[0] == ':' {
			continue
		}
		field, value := parseSSE(line)
		switch field {
		case "event":
			event = value
		case "data", "":
			data = append(data, value)
		default:
		}
	}
}

func (t *SSETransport) Write(data []byte) error {
	select {
	case <-t.postWait:
	default:
		return fmt.Errorf("%w: endpoint not discovered", ErrProtocol)
	}

	endpoint, err := resolveEndpoint(t.url, t.endpoint)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProtocol, err)
	}

	t.postMu.Lock()
	defer t.postMu.Unlock()

	postResp := make(chan []byte, 1)
	postError := make(chan error, 1)
	t.postResp = postResp
	t.postError = postError
	ready := make(chan struct{})

	go func() {
		close(ready)
		body := bytes.NewReader(data)
		req, err := http.NewRequest(http.MethodPost, endpoint, body)
		if err != nil {
			postError <- fmt.Errorf("mcp: sse build post: %w", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		if t.session != "" {
			req.Header.Set("Mcp-Session-Id", t.session)
		}
		for k, v := range t.headers {
			req.Header.Set(k, v)
		}

		resp, err := t.client.Do(req)
		if err != nil {
			postError <- fmt.Errorf("mcp: sse post: %w", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
			postError <- fmt.Errorf("mcp: sse post: HTTP %d: %s", resp.StatusCode, string(body))
			return
		}
		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			t.session = sid
		}
		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxFrameSize+1))
		if err != nil {
			postError <- fmt.Errorf("mcp: sse read post body: %w", err)
			return
		}
		if len(respBody) == 0 {
			postError <- nil
			return
		}
		postResp <- respBody
	}()

	<-ready

	select {
	case <-t.postWait:
	case <-postError:
		close(postResp)
		close(postError)
		return <-postError
	}

	select {
	case respBody := <-postResp:
		close(postResp)
		close(postError)
		if respBody == nil {
			return nil
		}
		t.readCh <- respBody
		return nil
	case err := <-postError:
		close(postResp)
		close(postError)
		return err
	}
}

func (t *SSETransport) ReadFrame() ([]byte, error) {
	select {
	case data := <-t.readCh:
		if data == nil {
			return nil, ErrClosed
		}
		return data, nil
	case err := <-t.errCh:
		if err == nil {
			return nil, ErrClosed
		}
		return nil, err
	}
}

func (t *SSETransport) Close() error {
	t.closeOnce.Do(func() {
		if t.resp != nil {
			_ = t.resp.Body.Close()
		}
		close(t.readCh)
	})
	return nil
}

func (t *SSETransport) fail(err error) {
	select {
	case t.errCh <- err:
	default:
	}
}

type sseLineReader struct {
	r       io.Reader
	readBuf []byte
	buf     []byte
}

func newSSELineReader(r io.Reader) *sseLineReader {
	return &sseLineReader{r: r, readBuf: make([]byte, 4096)}
}

func (l *sseLineReader) Next() ([]byte, error) {
	for {
		if i := bytes.Index(l.buf, []byte("\n")); i >= 0 {
			line := l.buf[:i]
			l.buf = l.buf[i+1:]
			return bytes.TrimSpace(line), nil
		}
		n, err := l.r.Read(l.readBuf)
		if n > 0 {
			l.buf = append(l.buf, l.readBuf[:n]...)
			continue
		}
		if err != nil {
			return nil, err
		}
		if len(l.buf) > 0 {
			line := l.buf
			l.buf = nil
			return bytes.TrimSpace(line), nil
		}
		return nil, io.EOF
	}
}

func resolveEndpoint(base, endpoint string) (string, error) {
	if endpoint == "" {
		return "", fmt.Errorf("empty endpoint")
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint, nil
	}
	if strings.HasPrefix(endpoint, "//") {
		if i := strings.Index(base, "://"); i >= 0 {
			return base[:i+3] + endpoint, nil
		}
		return "", fmt.Errorf("cannot resolve protocol-relative endpoint %q against %q", endpoint, base)
	}
	return url.JoinPath(base, endpoint)
}

func parseSSE(line []byte) (string, string) {
	i := bytes.IndexByte(line, ':')
	if i < 0 {
		return "", string(line)
	}
	field := string(line[:i])
	value := string(line[i+1:])
	if len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}
	return field, value
}
