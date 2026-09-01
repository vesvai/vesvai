package middleware

import "errors"

var (
	ErrNilMiddleware = errors.New("middleware: nil middleware")
	ErrEmptyName     = errors.New("middleware: empty name")
	ErrDuplicate     = errors.New("middleware: already registered")
	ErrNotFound      = errors.New("middleware: not found")
)
