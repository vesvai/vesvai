package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	done    bool
}

func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("sse: ResponseWriter does not support flushing")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	return &SSEWriter{w: w, flusher: flusher}, nil
}

func (s *SSEWriter) WriteEvent(event string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.done {
		return fmt.Errorf("sse: writer already closed")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sse: marshal data: %w", err)
	}

	_, err = fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, jsonData)
	if err != nil {
		return fmt.Errorf("sse: write event: %w", err)
	}

	s.flusher.Flush()
	return nil
}

func (s *SSEWriter) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.done = true
}

type SSEEvent struct {
	Type       string `json:"type"`
	AgentID    string `json:"agent_id,omitempty"`
	AgentName  string `json:"agent_name,omitempty"`
	Content    string `json:"content,omitempty"`
	Reasoning  string `json:"reasoning,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolArgs   string `json:"tool_args,omitempty"`
	ToolOutput string `json:"tool_output,omitempty"`
	Error      string `json:"error,omitempty"`
	Usage      any    `json:"usage,omitempty"`
	Strategy   string `json:"strategy,omitempty"`
	Messages   int    `json:"messages,omitempty"`
	Tokens     int    `json:"tokens,omitempty"`
	Done       bool   `json:"done,omitempty"`
}
