package session

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

func GetBuffer() *bytes.Buffer {
	return bufferPool.Get().(*bytes.Buffer)
}

func PutBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

func EncodeSession(session *Session) ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(session); err != nil {
		return nil, err
	}

	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

func DecodeSession(data []byte) (*Session, error) {
	var session Session
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}

type StreamDecoder struct {
	decoder *json.Decoder
}

func NewStreamDecoder(r io.Reader) *StreamDecoder {
	return &StreamDecoder{
		decoder: json.NewDecoder(r),
	}
}

func (d *StreamDecoder) Decode() (*Session, error) {
	var session Session
	if err := d.decoder.Decode(&session); err != nil {
		return nil, err
	}
	return &session, nil
}
