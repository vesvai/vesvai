package vfs

import "errors"

var (
	ErrOutOfBounds     = errors.New("vfs: path escapes the workspace root")
	ErrFileChanged     = errors.New("vfs: file changed externally, re-read required")
	ErrReadRequired    = errors.New("vfs: file must be read before editing")
	ErrIgnored         = errors.New("vfs: file not found or access denied")
	ErrNotFound        = errors.New("vfs: file not found")
	ErrNoMatch         = errors.New("vfs: pattern does not match file content")
	ErrInvalidGrepMode = errors.New("vfs: invalid grep output mode")
)
