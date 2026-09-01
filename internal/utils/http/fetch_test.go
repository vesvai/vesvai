package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchReturnsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected a default User-Agent")
		}
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	body, err := Fetch(context.Background(), srv.URL, FetchOptions{})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(body) != "hello world" {
		t.Fatalf("body = %q, want %q", body, "hello world")
	}
}

func TestFetchHonorsCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "yes" {
			t.Errorf("X-Test = %q, want yes", got)
		}
	}))
	defer srv.Close()

	if _, err := Fetch(context.Background(), srv.URL, FetchOptions{
		Headers: map[string]string{"X-Test": "yes"},
	}); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
}

func TestFetchCapsBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("0123456789"))
	}))
	defer srv.Close()

	body, err := Fetch(context.Background(), srv.URL, FetchOptions{MaxBytes: 5})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if string(body) != "01234" {
		t.Fatalf("body = %q, want %q", body, "01234")
	}
}

func TestFetchErrorOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("nope"))
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, FetchOptions{})
	if err == nil {
		t.Fatal("expected an error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error %q should mention status code", err)
	}
	var httpErr *HTTPError
	if !asHTTPError(err, &httpErr) {
		t.Fatalf("error should be an HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", httpErr.StatusCode)
	}
}

func TestFetchCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	if _, err := Fetch(ctx, srv.URL, FetchOptions{}); err == nil {
		t.Fatal("expected an error for a cancelled context")
	}
}

func asHTTPError(err error, target **HTTPError) bool {
	e, ok := err.(*HTTPError)
	if ok {
		*target = e
	}
	return ok
}
