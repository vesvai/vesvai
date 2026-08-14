package filesystem

import (
	"os"
	"regexp"
	"time"
)

type Config struct {
	RootDir        string
	IgnoreDotfiles bool
	MaxReadBytes   int64
}

type FileInfo struct {
	Path    string
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
}

type GrepResult struct {
	Path    string
	Line    int
	Content string
}

type IgnoreRule struct {
	Pattern string
	Regex   *regexp.Regexp
	Negate  bool
	DirOnly bool
	Source  string
}

type EditConflictError struct {
	Path string
}

func (e *EditConflictError) Error() string {
	return "file content has changed since last read, please re-read: " + e.Path
}
