package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchToolBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body><h1>Hello</h1><p>World</p></body></html>"))
	}))
	defer server.Close()

	tool := fetchTool(nil)
	out, err := tool.Execute(t.Context(), `{"url": "`+server.URL+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Hello") {
		t.Errorf("expected 'Hello' in output, got:\n%s", out)
	}
	if !contains(t, out, "World") {
		t.Errorf("expected 'World' in output, got:\n%s", out)
	}
	if !contains(t, out, "Status: 200") {
		t.Errorf("expected 'Status: 200', got:\n%s", out)
	}
}

func TestFetchToolPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	tool := fetchTool(nil)
	out, err := tool.Execute(t.Context(), `{"url": "`+server.URL+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "hello world") {
		t.Errorf("expected 'hello world', got:\n%s", out)
	}
}

func TestFetchToolMissingURL(t *testing.T) {
	tool := fetchTool(nil)
	_, err := tool.Execute(t.Context(), `{}`)
	if err == nil {
		t.Fatal("expected error for missing url")
	}
}

func TestFetchToolInvalidURL(t *testing.T) {
	tool := fetchTool(nil)
	_, err := tool.Execute(t.Context(), `{"url": "not-a-url"}`)
	if err == nil {
		t.Fatal("expected error for invalid url")
	}
}

func TestFetchToolStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<html><body>not found</body></html>"))
	}))
	defer server.Close()

	tool := fetchTool(nil)
	out, err := tool.Execute(t.Context(), `{"url": "`+server.URL+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Status: 404") {
		t.Errorf("expected 'Status: 404', got:\n%s", out)
	}
}

func TestFetchToolContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<p>test</p>"))
	}))
	defer server.Close()

	tool := fetchTool(nil)
	out, err := tool.Execute(t.Context(), `{"url": "`+server.URL+`"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "Content-Type: text/html") {
		t.Errorf("expected Content-Type info, got:\n%s", out)
	}
}

func TestFetchToolRawHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body><h1>Hello</h1><p>World</p></body></html>"))
	}))
	defer server.Close()

	tool := fetchTool(nil)
	out, err := tool.Execute(t.Context(), `{"url": "`+server.URL+`", "raw": true}`)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(t, out, "<h1>Hello</h1>") {
		t.Errorf("expected raw HTML '<h1>Hello</h1>', got:\n%s", out)
	}
	if !contains(t, out, "<html>") {
		t.Errorf("expected raw HTML '<html>', got:\n%s", out)
	}
}

func TestIsHTML(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"<!DOCTYPE html><html>", true},
		{"<html><body>", true},
		{"<html>", true},
		{"plain text", false},
		{"just some text", false},
	}
	for _, tt := range tests {
		got := isHTML([]byte(tt.input))
		if got != tt.want {
			t.Errorf("isHTML(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func contains(t *testing.T, s, substr string) bool {
	t.Helper()
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}