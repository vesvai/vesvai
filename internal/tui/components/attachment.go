package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AttachmentType int

const (
	AttachmentText AttachmentType = iota
	AttachmentImage
	AttachmentVideo
	AttachmentDocument
	AttachmentUnknown
)

type Attachment struct {
	Type        AttachmentType
	Name        string
	Content     string
	FilePath    string
	Size        int64
	MimeType    string
	Previewable bool
}

var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".go": true, ".py": true, ".js": true,
	".ts": true, ".jsx": true, ".tsx": true, ".html": true, ".css": true,
	".json": true, ".xml": true, ".yaml": true, ".yml": true, ".toml": true,
	".sh": true, ".bash": true, ".zsh": true, ".rs": true, ".java": true,
	".c": true, ".cpp": true, ".h": true, ".hpp": true, ".rb": true,
	".php": true, ".swift": true, ".kt": true, ".scala": true, ".conf": true,
	".cfg": true, ".ini": true, ".env": true, ".gitignore": true,
	".dockerfile": true, ".makefile": true, ".log": true, ".csv": true,
}

var imageExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true,
	".svg": true, ".webp": true, ".ico": true, ".tiff": true,
}

var videoExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mov": true, ".mkv": true, ".webm": true,
	".flv": true, ".wmv": true,
}

var documentExtensions = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".odt": true, ".ods": true, ".odp": true,
	".rtf": true,
}

func NewTextAttachment(content string) *Attachment {
	name := "pasted text"
	if len(content) > 30 {
		name = content[:30] + "..."
	}
	return &Attachment{
		Type:        AttachmentText,
		Name:        name,
		Content:     content,
		Size:        int64(len(content)),
		MimeType:    "text/plain",
		Previewable: true,
	}
}

func NewFileAttachment(filePath string) (*Attachment, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot access file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	attType, previewable := detectType(ext)
	mime := mimeFromExt(ext)

	att := &Attachment{
		Type:        attType,
		Name:        filepath.Base(filePath),
		FilePath:    filePath,
		Size:        info.Size(),
		MimeType:    mime,
		Previewable: previewable,
	}

	if previewable {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("cannot read file: %w", err)
		}
		att.Content = string(data)
	}

	return att, nil
}

func detectType(ext string) (AttachmentType, bool) {
	if imageExtensions[ext] {
		return AttachmentImage, false
	}
	if videoExtensions[ext] {
		return AttachmentVideo, false
	}
	if documentExtensions[ext] {
		return AttachmentDocument, false
	}
	if textExtensions[ext] {
		return AttachmentText, true
	}
	return AttachmentUnknown, false
}

func mimeFromExt(ext string) string {
	mimes := map[string]string{
		".txt": "text/plain", ".md": "text/markdown", ".go": "text/x-go",
		".py": "text/x-python", ".js": "text/javascript", ".ts": "text/typescript",
		".html": "text/html", ".css": "text/css", ".json": "application/json",
		".xml": "application/xml", ".yaml": "text/yaml", ".yml": "text/yaml",
		".sh": "text/x-shellscript", ".rs": "text/x-rust", ".java": "text/x-java",
		".c": "text/x-c", ".cpp": "text/x-c++", ".h": "text/x-c-header",
		".rb": "text/x-ruby", ".php": "text/x-php",
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".svg": "image/svg+xml", ".webp": "image/webp",
		".mp4": "video/mp4", ".avi": "video/x-msvideo", ".mov": "video/quicktime",
		".mkv": "video/x-matroska", ".webm": "video/webm",
		".pdf": "application/pdf", ".doc": "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}
	if m, ok := mimes[ext]; ok {
		return m
	}
	return "application/octet-stream"
}

func (a *Attachment) Icon() string {
	switch a.Type {
	case AttachmentText:
		return "✦"
	case AttachmentImage:
		return "◆"
	case AttachmentVideo:
		return "▶"
	case AttachmentDocument:
		return "→"
	default:
		return "●"
	}
}

func (a *Attachment) ShortName(maxLen int) string {
	runes := []rune(a.Name)
	if len(runes) <= maxLen {
		return a.Name
	}
	return string(runes[:maxLen-1]) + "…"
}

func (a *Attachment) SizeFormatted() string {
	if a.Size < 1024 {
		return fmt.Sprintf("%d B", a.Size)
	}
	if a.Size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(a.Size)/1024)
	}
	if a.Size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(a.Size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(a.Size)/(1024*1024*1024))
}

func (a *Attachment) TypeLabel() string {
	switch a.Type {
	case AttachmentText:
		return "Text"
	case AttachmentImage:
		return "Image"
	case AttachmentVideo:
		return "Video"
	case AttachmentDocument:
		return "Document"
	default:
		return "File"
	}
}

func LookupFileFromText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	if trimmed[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			trimmed = home + trimmed[1:]
		}
	}

	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "./") || strings.HasPrefix(trimmed, "../") {
		if _, err := os.Stat(trimmed); err == nil {
			return trimmed
		}
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(cwd, trimmed)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
