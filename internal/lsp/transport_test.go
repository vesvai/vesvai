package lsp

import (
	"bytes"
	json "github.com/goccy/go-json"
	"strconv"
	"strings"
	"testing"
)

func TestStdioSpawnFailure(t *testing.T) {
	_, err := NewStdioTransport(StdioOptions{Command: "definitely-not-a-real-command-xyz"})
	if err == nil {
		t.Fatal("expected spawn error")
	}
}

func TestStdioRoundTrip(t *testing.T) {
	tr, err := NewStdioTransport(StdioOptions{
		Command: "cat",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	payload, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})
	if err := tr.Write(payload); err != nil {
		t.Fatal(err)
	}
	frame, err := tr.ReadFrame()
	if err != nil {
		t.Fatal(err)
	}
	var echoed map[string]any
	if err := json.Unmarshal(frame, &echoed); err != nil {
		t.Fatal(err)
	}
	if echoed["id"] != float64(1) {
		t.Fatalf("echoed = %+v", echoed)
	}
}

func TestStdioCloseKillsProcess(t *testing.T) {
	tr, err := NewStdioTransport(StdioOptions{Command: "sleep", Args: []string{"30"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tr.Write([]byte(`{}`)); err == nil {
		t.Fatal("expected write after close to fail")
	}
}

func TestFrameDecoderContentLength(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":{}}`
	frame := "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
	dec := NewFrameDecoder(bytes.NewReader([]byte(frame)))
	out, err := dec.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != body {
		t.Fatalf("decoded = %q, want %q", string(out), body)
	}
}

func TestFrameDecoderNewlineFallback(t *testing.T) {
	dec := NewFrameDecoder(bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1}\n`)))
	out, err := dec.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "jsonrpc") == false {
		t.Fatalf("decoded = %q", string(out))
	}
}
