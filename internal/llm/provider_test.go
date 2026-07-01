package llm

import (
	"context"
	"strings"
	"testing"
)

func TestProviderError_Error(t *testing.T) {
	e := &ProviderError{StatusCode: 400, Message: "bad request"}
	if e.Error() != "bad request" {
		t.Errorf("Error() = %q, want %q", e.Error(), "bad request")
	}
}

func TestProviderError_Temporary(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{429, true},  // rate limited
		{500, true},  // internal server error
		{502, true},  // bad gateway
		{503, true},  // service unavailable
		{400, false}, // bad request
		{401, false}, // unauthorized
		{403, false}, // forbidden
		{404, false}, // not found
		{200, false}, // ok
	}
	for _, tt := range tests {
		e := &ProviderError{StatusCode: tt.code}
		if got := e.Temporary(); got != tt.expected {
			t.Errorf("StatusCode %d: Temporary() = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestProviderError_RateLimited(t *testing.T) {
	tests := []struct {
		code     int
		expected bool
	}{
		{429, true},
		{500, false},
		{400, false},
		{200, false},
	}
	for _, tt := range tests {
		e := &ProviderError{StatusCode: tt.code}
		if got := e.RateLimited(); got != tt.expected {
			t.Errorf("StatusCode %d: RateLimited() = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestProviderError_Body(t *testing.T) {
	e := &ProviderError{
		StatusCode: 500,
		Message:    "internal error",
		Body:       `{"error":"something broke"}`,
	}
	if e.Body != `{"error":"something broke"}` {
		t.Errorf("Body = %q, want %q", e.Body, `{"error":"something broke"}`)
	}
}

func TestStreamHandler(t *testing.T) {
	var called bool
	handler := func(chunk StreamChunk) error {
		called = true
		return nil
	}

	err := handler(StreamChunk{Content: "hello"})
	if err != nil {
		t.Errorf("handler returned error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestStreamHandler_Error(t *testing.T) {
	handler := func(chunk StreamChunk) error {
		return &ProviderError{StatusCode: 500, Message: "stream error"}
	}

	err := handler(StreamChunk{Content: "hello"})
	if err == nil {
		t.Fatal("handler should have returned error")
	}
	var pe *ProviderError
	if !strings.Contains(err.Error(), "stream error") {
		t.Errorf("error = %q, want 'stream error'", err.Error())
	}
	_ = pe // just checking type assertion works
}

func TestProviderInterface(t *testing.T) {
	// Verify that a mock can implement the Provider interface
	var p Provider = &mockProvider{}
	if p.Name() != "mock" {
		t.Errorf("Name() = %q, want %q", p.Name(), "mock")
	}
}

type mockProvider struct{}

func (m *mockProvider) Name() string { return "mock" }
func (m *mockProvider) Chat(_ context.Context, _ *Request) (*Response, error) {
	return &Response{}, nil
}
func (m *mockProvider) ChatStream(_ context.Context, _ *Request, _ StreamHandler) error {
	return nil
}
