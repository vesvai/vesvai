package filesystem

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const (
	defaultMaxReadBytes = 16 << 20
	maxGrepFileBytes    = 8 << 20
	maxChecksums        = 4096
)

type FileSystem struct {
	root          string
	ignore        *IgnoreRules
	ignoreDot     bool
	maxReadBytes  int64
	mu            sync.RWMutex
	readChecksums map[string]string
}

func New(cfg Config) (*FileSystem, error) {
	absRoot, err := filepath.Abs(cfg.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve root dir: %w", err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("root dir does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root path is not a directory: %s", absRoot)
	}

	ignore, err := LoadIgnoreRules(absRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load ignore rules: %w", err)
	}

	maxRead := cfg.MaxReadBytes
	if maxRead <= 0 {
		maxRead = defaultMaxReadBytes
	}

	return &FileSystem{
		root:          absRoot,
		ignore:        ignore,
		ignoreDot:     cfg.IgnoreDotfiles,
		maxReadBytes:  maxRead,
		readChecksums: make(map[string]string),
	}, nil
}

func (fs *FileSystem) Root() string {
	return fs.root
}

func (fs *FileSystem) Resolve(relPath string) (string, error) {
	relPath = cleanPath(relPath)

	if strings.HasPrefix(relPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home dir: %w", err)
		}
		relPath = filepath.Join(home, relPath[2:])
	} else if relPath == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot resolve home dir: %w", err)
		}
		relPath = home
	}

	if strings.Contains(relPath, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", relPath)
	}

	absPath := filepath.Join(fs.root, relPath)

	resolved, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	if !strings.HasPrefix(resolved, fs.root) {
		return "", fmt.Errorf("path escapes root directory: %s", relPath)
	}

	return resolved, nil
}

func (fs *FileSystem) IsIgnored(relPath string) bool {
	relPath = cleanPath(relPath)
	info, err := os.Stat(filepath.Join(fs.root, relPath))
	isDir := false
	if err == nil {
		isDir = info.IsDir()
	}
	return fs.shouldInclude(relPath, isDir) == false
}

func (fs *FileSystem) Stat(relPath string) (FileInfo, error) {
	relPath = cleanPath(relPath)

	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return FileInfo{}, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to stat file: %w", err)
	}

	return FileInfo{
		Path:    relPath,
		Name:    info.Name(),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
		Mode:    info.Mode(),
	}, nil
}

func (fs *FileSystem) Remove(relPath string) error {
	relPath = cleanPath(relPath)

	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return err
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}

	return nil
}

func (fs *FileSystem) shouldInclude(relPath string, isDir bool) bool {
	relPath = cleanPath(relPath)

	if fs.ignore.IsIgnored(relPath, isDir) {
		return false
	}

	if fs.ignoreDot && isDotPath(relPath) {
		return false
	}

	return true
}

func (fs *FileSystem) Read(ctx context.Context, relPath string) (string, error) {
	relPath = cleanPath(relPath)

	if !fs.shouldInclude(relPath, false) {
		return "", fmt.Errorf("access denied: file is ignored or hidden: %s", relPath)
	}

	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return "", err
	}

	data, truncated, err := fs.readCapped(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	if IsBinary(data) {
		return fmt.Sprintf("[Binary file: %s — cannot display content]", filepath.Base(relPath)), nil
	}

	if truncated {
		fs.removeChecksum(relPath)
		return string(data) + "\n...[truncated: file exceeds read limit]", nil
	}

	fs.storeChecksum(relPath, data)
	return string(data), nil
}

func (fs *FileSystem) readCapped(absPath string) ([]byte, bool, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, fs.maxReadBytes+1))
	if err != nil {
		return nil, false, err
	}

	if int64(len(data)) > fs.maxReadBytes {
		return data[:fs.maxReadBytes], true, nil
	}
	return data, false, nil
}

func (fs *FileSystem) storeChecksum(relPath string, data []byte) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if len(fs.readChecksums) >= maxChecksums {
		fs.readChecksums = make(map[string]string)
	}
	fs.readChecksums[relPath] = hashFromBytes(data)
}

func (fs *FileSystem) removeChecksum(relPath string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	delete(fs.readChecksums, relPath)
}

func (fs *FileSystem) ReadRange(ctx context.Context, relPath string, start, end int) (string, error) {
	content, err := fs.Read(ctx, relPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(content, "\n")
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > len(lines) {
		return "", fmt.Errorf("start line %d exceeds file length %d", start, len(lines))
	}

	selected := lines[start-1 : end]
	var result strings.Builder
	for i, line := range selected {
		fmt.Fprintf(&result, "%d: %s\n", start+i, line)
	}

	return result.String(), nil
}

func (fs *FileSystem) Write(ctx context.Context, relPath string, content []byte) error {
	relPath = cleanPath(relPath)

	if !fs.shouldInclude(relPath, false) {
		return fmt.Errorf("access denied: cannot write to ignored or hidden file: %s", relPath)
	}

	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return err
	}

	fs.mu.RLock()
	oldHash, wasRead := fs.readChecksums[relPath]
	fs.mu.RUnlock()

	if wasRead {
		currentHash, err := hashFileContent(absPath)
		if err == nil && currentHash != oldHash {
			return &EditConflictError{Path: relPath}
		}
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(absPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fs.mu.Lock()
	fs.readChecksums[relPath] = hashFromBytes(content)
	fs.mu.Unlock()

	return nil
}

func (fs *FileSystem) Edit(ctx context.Context, relPath, oldString, newString string, expectedLine int) error {
	relPath = cleanPath(relPath)

	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return err
	}

	fs.mu.RLock()
	oldHash, wasRead := fs.readChecksums[relPath]
	fs.mu.RUnlock()

	if wasRead {
		currentHash, err := hashFileContent(absPath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if currentHash != oldHash {
			return &EditConflictError{Path: relPath}
		}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	if expectedLine > 0 {
		if expectedLine > len(lines) {
			return fmt.Errorf("line %d exceeds file length %d", expectedLine, len(lines))
		}

		idx := expectedLine - 1
		if !strings.Contains(lines[idx], oldString) {
			return fmt.Errorf("old string not found at line %d", expectedLine)
		}
		lines[idx] = strings.Replace(lines[idx], oldString, newString, 1)
	} else {
		found := false
		for i, line := range lines {
			if strings.Contains(line, oldString) {
				lines[i] = strings.Replace(line, oldString, newString, 1)
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("old string not found in file")
		}
	}

	newContent := strings.Join(lines, "\n")
	return fs.writeUnchecked(relPath, []byte(newContent))
}

func (fs *FileSystem) writeUnchecked(relPath string, content []byte) error {
	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(absPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fs.mu.Lock()
	fs.readChecksums[relPath] = hashFromBytes(content)
	fs.mu.Unlock()

	return nil
}

func (fs *FileSystem) List(ctx context.Context, relPath string) ([]FileInfo, error) {
	relPath = cleanPath(relPath)
	if relPath == "" {
		relPath = "."
	}

	if !fs.shouldInclude(relPath, true) {
		return nil, fmt.Errorf("access denied: directory is ignored or hidden: %s", relPath)
	}

	absPath, err := fs.Resolve(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []FileInfo
	for _, entry := range entries {
		name := entry.Name()
		entryRel := cleanPath(filepath.Join(relPath, name))

		info, err := entry.Info()
		if err != nil {
			continue
		}

		isDir := entry.IsDir()
		if !fs.shouldInclude(entryRel, isDir) {
			continue
		}

		result = append(result, FileInfo{
			Path:    entryRel,
			Name:    name,
			IsDir:   isDir,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].IsDir != result[j].IsDir {
			return result[i].IsDir
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}

func (fs *FileSystem) Glob(ctx context.Context, pattern string) ([]FileInfo, error) {
	pattern = cleanPath(pattern)
	if pattern == "" || pattern == "." {
		pattern = "**"
	}
	if !strings.Contains(pattern, "/") {
		pattern = "**/" + pattern
	}

	patSegs := strings.Split(pattern, "/")

	baseDir := filepath.Join(fs.root, globBaseDir(pattern))

	var result []FileInfo
	err := filepath.WalkDir(baseDir, func(path string, d iofs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, err := filepath.Rel(fs.root, path)
		if err != nil {
			return nil
		}
		rel = cleanPath(rel)

		if d.IsDir() {
			if rel != "" && rel != "." && !fs.shouldInclude(rel, true) {
				return filepath.SkipDir
			}
			return nil
		}

		if !fs.shouldInclude(rel, false) {
			return nil
		}

		relSegs := strings.Split(rel, "/")
		if !matchGlobSegs(patSegs, relSegs) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		result = append(result, FileInfo{
			Path:    rel,
			Name:    info.Name(),
			IsDir:   false,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("glob walk: %w", err)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ModTime.After(result[j].ModTime)
	})

	return result, nil
}

func globBaseDir(pattern string) string {
	segs := strings.Split(pattern, "/")
	for i, s := range segs {
		if strings.ContainsAny(s, "*?[") {
			return strings.Join(segs[:i], "/")
		}
	}
	return ""
}

func matchGlobSegs(pat, name []string) bool {
	if len(pat) == 0 {
		return len(name) == 0
	}
	if pat[0] == "**" {
		for i := 0; i <= len(name); i++ {
			if matchGlobSegs(pat[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, err := filepath.Match(pat[0], name[0])
	if err != nil || !ok {
		return false
	}
	return matchGlobSegs(pat[1:], name[1:])
}

func (fs *FileSystem) Walk(ctx context.Context, fn func(relPath string, info FileInfo) error) error {
	return filepath.Walk(fs.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(fs.root, path)
		if err != nil {
			return nil
		}
		relPath = cleanPath(relPath)

		if relPath == "." {
			return nil
		}

		if !fs.shouldInclude(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return fn(relPath, FileInfo{
			Path:    relPath,
			Name:    info.Name(),
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		})
	})
}

func (fs *FileSystem) Grep(ctx context.Context, pattern string, paths ...string) ([]GrepResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid grep pattern: %w", err)
	}

	if len(paths) == 0 {
		paths = []string{"."}
	}

	var results []GrepResult

	for _, searchPath := range paths {
		searchPath = cleanPath(searchPath)
		absPath, err := fs.Resolve(searchPath)
		if err != nil {
			return nil, err
		}

		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			fs.walkForGrep(absPath, searchPath, re, &results)
		} else {
			if fs.shouldInclude(searchPath, false) {
				fs.grepFile(absPath, searchPath, re, &results)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Path != results[j].Path {
			return results[i].Path < results[j].Path
		}
		return results[i].Line < results[j].Line
	})

	return results, nil
}

func (fs *FileSystem) walkForGrep(absDir, relDir string, re *regexp.Regexp, results *[]GrepResult) {
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		name := entry.Name()
		entryRel := cleanPath(filepath.Join(relDir, name))
		entryAbs := filepath.Join(absDir, name)

		if !fs.shouldInclude(entryRel, entry.IsDir()) {
			if entry.IsDir() {
				continue
			}
			continue
		}

		if entry.IsDir() {
			fs.walkForGrep(entryAbs, entryRel, re, results)
		} else {
			fs.grepFile(entryAbs, entryRel, re, results)
		}
	}
}

func (fs *FileSystem) grepFile(absPath, relPath string, re *regexp.Regexp, results *[]GrepResult) {
	f, err := os.Open(absPath)
	if err != nil {
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return
	}
	if fi.Size() > maxGrepFileBytes {
		return
	}

	head := make([]byte, 512)
	n, _ := f.Read(head)
	if IsBinary(head[:n]) {
		return
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			*results = append(*results, GrepResult{
				Path:    relPath,
				Line:    lineNum,
				Content: line,
			})
		}
	}
}

func hashFromBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
