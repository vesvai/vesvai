package mcp

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"testing"
)

func decodeAll(t *testing.T, d *FrameDecoder, want []string) {
	for _, w := range want {
		got, err := d.Decode()
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != w {
			t.Fatalf("frame = %q, want %q", string(got), w)
		}
	}
}

func TestFrameDecoderNewline(t *testing.T) {
	input := []byte("{\"a\":1}\n{\"b\":2}\n")
	r := bytes.NewReader(input)
	d := NewFrameDecoder(r)
	decodeAll(t, d, []string{`{"a":1}`, `{"b":2}`})
}

func TestFrameDecoderContentLength(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":{}}`
	frame := "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
	r := bytes.NewReader([]byte(frame))
	d := NewFrameDecoder(r)
	decodeAll(t, d, []string{body})
}

func TestFrameDecoderChunkedContentLength(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`
	header := "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n"
	all := []byte(header + body)

	d := NewFrameDecoder(&chunkReader{data: all, size: 1})
	got, err := d.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("frame = %q, want %q", string(got), body)
	}
	if _, err := d.Decode(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}

type chunkReader struct {
	data []byte
	pos  int
	size int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if c.pos >= len(c.data) {
		return 0, io.EOF
	}
	n := c.size
	if c.pos+n > len(c.data) {
		n = len(c.data) - c.pos
	}
	copy(p, c.data[c.pos:c.pos+n])
	c.pos += n
	return n, nil
}

func TestFrameDecoderMixedFraming(t *testing.T) {
	body := `{"a":1}`
	frame := "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body + "\n"
	r := bytes.NewReader([]byte(frame + `{"b":2}` + "\n"))
	d := NewFrameDecoder(r)
	decodeAll(t, d, []string{body, `{"b":2}`})
}

func TestFrameDecoderEOF(t *testing.T) {
	r := bytes.NewReader([]byte(`{"a":1}`))
	d := NewFrameDecoder(r)
	got, err := d.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("frame = %q", string(got))
	}
	if _, err := d.Decode(); !errors.Is(err, io.EOF) {
		t.Fatalf("expected EOF, got %v", err)
	}
}
