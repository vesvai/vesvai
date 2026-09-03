package mcp

import (
	json "github.com/goccy/go-json"
	"testing"
)

func TestBuildRequest(t *testing.T) {
	data, err := buildRequest(7, MethodPing, nil)
	if err != nil {
		t.Fatal(err)
	}
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatal(err)
	}
	if req.JSONRPC != "2.0" || req.ID != 7 || req.Method != MethodPing {
		t.Fatalf("bad request: %s", string(data))
	}
}

func TestBuildRequestWithParams(t *testing.T) {
	data, err := buildRequest(1, MethodInitialize, map[string]any{"protocolVersion": ProtocolVersion})
	if err != nil {
		t.Fatal(err)
	}
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatal(err)
	}
	if len(req.Params) == 0 {
		t.Fatal("params missing")
	}
	var parsed any
	if err := json.Unmarshal(req.Params, &parsed); err != nil {
		t.Fatal(err)
	}
	m, ok := parsed.(map[string]any)
	if !ok || m["protocolVersion"] != ProtocolVersion {
		t.Fatalf("params = %+v", parsed)
	}
}

func TestBuildNotification(t *testing.T) {
	data, err := buildNotification(MethodInitialized, nil)
	if err != nil {
		t.Fatal(err)
	}
	var n Notification
	if err := json.Unmarshal(data, &n); err != nil {
		t.Fatal(err)
	}
	if n.JSONRPC != "2.0" || n.Method != MethodInitialized || len(n.Params) != 0 {
		t.Fatalf("bad notification: %s", string(data))
	}
}

func TestParseResponseResult(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":3,"result":{"tools":[]}}`
	resp, err := parseResponse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != 3 {
		t.Fatalf("id = %d, want 3", resp.ID)
	}
	var lr ListToolsResult
	if err := json.Unmarshal(resp.Result, &lr); err != nil {
		t.Fatal(err)
	}
	if len(lr.Tools) != 0 {
		t.Fatalf("tools = %+v", lr.Tools)
	}
}

func TestParseResponseError(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":9,"error":{"code":-32601,"message":"method not found"}}`
	resp, err := parseResponse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("error = %+v", resp.Error)
	}
}

func TestIsNotification(t *testing.T) {
	if !isNotification([]byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)) {
		t.Fatal("should be a notification")
	}
	if isNotification([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`)) {
		t.Fatal("should be a response")
	}
}
