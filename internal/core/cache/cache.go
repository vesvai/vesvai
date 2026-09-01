package cache

import "errors"

var ErrNotFound = errors.New("cache: key not found")

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	Clear() error
	Close() error
}
