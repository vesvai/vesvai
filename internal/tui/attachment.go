package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Attachment struct {
	Path string
	Name string
	Kind string
	Size int64
}

func DetectAttachment(path string) (*Attachment, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, false
	}
	return &Attachment{
		Path: path,
		Name: filepath.Base(path),
		Kind: AttachmentKind(path),
		Size: info.Size(),
	}, true
}

func AttachmentKind(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".ico", ".avif":
		return "image"
	case ".mp4", ".mov", ".webm", ".mkv", ".avi", ".m4v":
		return "video"
	case ".pdf":
		return "pdf"
	default:
		return "file"
	}
}

func FormatSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
