package http

import (
	"bytes"
	"io"
)

type LineDecoder struct {
	reader *io.Reader
	buffer []byte
}

func NewLineDecoder(r io.Reader) *LineDecoder {
	return &LineDecoder{
		reader: &r,
		buffer: make([]byte, 0),
	}
}

func (ld *LineDecoder) Decode() ([]byte, error) {
	for {
		buf := make([]byte, 1024)
		n, err := (*ld.reader).Read(buf)
		if n > 0 {
			ld.buffer = append(ld.buffer, buf[:n]...)
		}

		if i := bytes.Index(ld.buffer, []byte("\n")); i >= 0 {
			line := ld.buffer[:i]
			ld.buffer = ld.buffer[i+1:]
			return bytes.TrimSpace(line), nil
		}

		if err != nil {
			if len(ld.buffer) > 0 {
				line := ld.buffer
				ld.buffer = nil
				return bytes.TrimSpace(line), nil
			}
			return nil, err
		}
	}
}

func ParseSSEvent(line []byte) (string, string) {
	if len(line) < 6 {
		return "", ""
	}
	if !bytes.HasPrefix(line, []byte("data:")) {
		return "", ""
	}

	data := bytes.TrimSpace(line[5:])

	if bytes.Equal(data, []byte("[DONE]")) {
		return "done", ""
	}

	return "data", string(data)
}
