package vfs

import (
	"errors"
	"fmt"
	"path/filepath"
)

var (
	ErrOutOfBounds     = errors.New("vfs: path escapes the workspace root")
	ErrFileChanged     = errors.New("vfs: file changed externally, re-read required")
	ErrReadRequired    = errors.New("vfs: file must be read before editing")
	ErrIgnored         = errors.New("vfs: file not found or access denied")
	ErrNotFound        = errors.New("vfs: file not found")
	ErrNoMatch         = errors.New("vfs: pattern does not match file content")
	ErrMultipleMatch   = errors.New("vfs: multiple files match the pattern")
	ErrInvalidGrepMode = errors.New("vfs: invalid grep output mode")
)

type OutOfBoundsError struct {
	Path string
}

func (e *OutOfBoundsError) Error() string {
	return fmt.Sprintf("%s: %s", ErrOutOfBounds, e.Path)
}

func (e *OutOfBoundsError) Unwrap() error {
	return ErrOutOfBounds
}

func (v *VFS) outOfBounds(path string) error {
	if path == "" {
		return ErrOutOfBounds
	}
	return &OutOfBoundsError{Path: filepath.Clean(path)}
}
