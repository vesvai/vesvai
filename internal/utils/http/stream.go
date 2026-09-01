package http

import (
	"bytes"
	"errors"
	"io"
)

const maxLineSize = 1 << 26

type LineDecoder struct {
	reader    io.Reader
	buffer    []byte
	readBuf   []byte
	searchPos int
}

func NewLineDecoder(r io.Reader) *LineDecoder {
	return &LineDecoder{
		reader:  r,
		buffer:  make([]byte, 0, 1024),
		readBuf: make([]byte, 1024),
	}
}

func (ld *LineDecoder) Decode() ([]byte, error) {
	for {
		if i := bytes.IndexByte(ld.buffer[ld.searchPos:], '\n'); i >= 0 {
			idx := ld.searchPos + i
			line := ld.buffer[:idx]
			ld.buffer = ld.buffer[idx+1:]
			ld.searchPos = 0
			return bytes.TrimSpace(line), nil
		}
		ld.searchPos = len(ld.buffer)

		n, err := ld.reader.Read(ld.readBuf)
		if n > 0 {
			ld.buffer = append(ld.buffer, ld.readBuf[:n]...)
			if len(ld.buffer) > maxLineSize {
				return nil, errors.New("stream: line exceeds maximum size")
			}
		}

		if err != nil {
			if len(ld.buffer) > 0 {
				line := ld.buffer
				ld.buffer = nil
				ld.searchPos = 0
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
