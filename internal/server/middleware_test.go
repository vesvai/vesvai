package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthMiddlewareNoRequiredHeaders(t *testing.T) {
	srv := newTestServer(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when no required headers, got %d", w.Code)
	}
}

func TestAuthMiddlewareMissingHeader(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"Authorization": ""}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when header missing, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "Authorization") {
		t.Errorf("expected error mentioning Authorization, got %s", resp.Error)
	}
}

func TestAuthMiddlewareInvalidHeader(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"Authorization": "Bearer expected"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when header invalid, got %d", w.Code)
	}

	var resp ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "invalid header") {
		t.Errorf("expected error about invalid header, got %s", resp.Error)
	}
}

func TestAuthMiddlewareValidHeader(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"Authorization": "Bearer expected"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer expected")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when header valid, got %d", w.Code)
	}
}

func TestAuthMiddlewareCaseInsensitive(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"Authorization": "Bearer secret"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("authorization", "bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with case-insensitive match, got %d", w.Code)
	}
}

func TestAuthMiddlewarePresenceOnly(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"X-API-Key": ""}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "any-value")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 when header present (no value check), got %d", w.Code)
	}
}

func TestAuthMiddlewareMultipleHeaders(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{
		"Authorization": "Bearer secret",
		"X-API-Version": "v1",
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 missing second header, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-API-Version", "v1")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with both headers, got %d", w.Code)
	}
}

func TestAuthMiddlewareWithEmptyValue(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"X-API-Key": ""}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "anything")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with any value, got %d", w.Code)
	}
}

func TestCorsMiddleware(t *testing.T) {
	srv := newTestServer(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.corsMiddleware(next)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}
}

func TestCorsMiddlewareOptions(t *testing.T) {
	srv := newTestServer(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for OPTIONS")
	})

	handler := srv.corsMiddleware(next)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

func TestCorsMiddlewareCustomHeaders(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.Headers = map[string]string{
		"X-Custom-Header": "custom-value",
		"X-API-Version":   "v2",
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.corsMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header custom-value, got %s", w.Header().Get("X-Custom-Header"))
	}
	if w.Header().Get("X-API-Version") != "v2" {
		t.Errorf("expected X-API-Version v2, got %s", w.Header().Get("X-API-Version"))
	}
}

func TestLoggingMiddleware(t *testing.T) {
	srv := newTestServer(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.loggingMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestLoggingMiddlewareRecordsStatus(t *testing.T) {
	srv := newTestServer(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := srv.loggingMiddleware(next)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected statusCode 201, got %d", rw.statusCode)
	}
}

func TestMiddlewareChain(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"Authorization": "Bearer test"}
	srv.cfg.Server.Headers = map[string]string{"X-Custom": "value"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := srv.authMiddleware(srv.corsMiddleware(next))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer test")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Custom") != "value" {
		t.Errorf("expected X-Custom value, got %s", w.Header().Get("X-Custom"))
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestMiddlewareChainAuthFails(t *testing.T) {
	srv := newTestServer(t)
	srv.cfg.Server.RequiredHeaders = map[string]string{"Authorization": "Bearer test"}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called when auth fails")
	})

	handler := srv.authMiddleware(srv.corsMiddleware(next))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
