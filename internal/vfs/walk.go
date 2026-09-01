package vfs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Entry struct {
	Name  string
	Path  string
	Size  int64
	IsDir bool
}

type ListResult struct {
	Path      string
	Entries   []Entry
	FileCount int
	DirCount  int
	TotalSize int64
}

func (v *VFS) List(vdir string) (ListResult, error) {
	phys, rel, isDir, err := v.resolveChecked(vdir)
	if err != nil {
		return ListResult{}, err
	}
	if !isDir {
		return ListResult{}, ErrNotFound
	}

	entries, err := os.ReadDir(phys)
	if err != nil {
		return ListResult{}, err
	}

	res := ListResult{Path: rel, Entries: make([]Entry, 0, len(entries))}
	for _, e := range entries {
		vrel := joinVirtual(rel, e.Name())
		if v.ignored(vrel, e.IsDir()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if e.IsDir() {
			res.DirCount++
		} else {
			res.FileCount++
			res.TotalSize += info.Size()
		}
		res.Entries = append(res.Entries, Entry{
			Name:  e.Name(),
			Path:  vrel,
			Size:  info.Size(),
			IsDir: e.IsDir(),
		})
	}
	sort.Slice(res.Entries, func(i, j int) bool { return res.Entries[i].Name < res.Entries[j].Name })
	return res, nil
}

func (v *VFS) Glob(pattern, path string) ([]string, error) {
	if path == "" {
		path = "."
	}
	basePhys, baseRel, isDir, err := v.resolveChecked(path)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, ErrNotFound
	}

	pattern = filepath.ToSlash(pattern)
	if pattern == "" || strings.HasPrefix(pattern, "/") || strings.ContainsRune(pattern, 0) {
		return nil, ErrOutOfBounds
	}
	for _, seg := range strings.Split(pattern, "/") {
		if seg == ".." {
			return nil, ErrOutOfBounds
		}
	}

	var out []string
	seen := make(map[string]bool)
	add := func(rel string) {
		if !seen[rel] {
			seen[rel] = true
			out = append(out, rel)
		}
	}

	segs := strings.Split(pattern, "/")
	anchored := len(segs) > 1
	if err := v.globWalk(basePhys, baseRel, segs, anchored, add); err != nil {
		return nil, err
	}

	mods := make(map[string]time.Time, len(out))
	for _, rel := range out {
		if fi, err := os.Stat(filepath.Join(v.root, filepath.FromSlash(rel))); err == nil {
			mods[rel] = fi.ModTime()
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := mods[out[i]], mods[out[j]]
		if !a.Equal(b) {
			return a.After(b)
		}
		return out[i] < out[j]
	})
	return out, nil
}

func (v *VFS) globWalk(physDir, vrel string, segs []string, anchored bool, add func(string)) error {
	if len(segs) == 0 {
		return nil
	}

	if segs[0] == "**" {
		if err := v.globWalk(physDir, vrel, segs[1:], anchored, add); err != nil {
			return err
		}
		if len(segs) == 1 {
			entries, err := os.ReadDir(physDir)
			if err != nil {
				return nil
			}
			for _, e := range entries {
				childRel := joinVirtual(vrel, e.Name())
				if v.ignored(childRel, e.IsDir()) {
					continue
				}
				if e.IsDir() {
					if err := v.globWalk(filepath.Join(physDir, e.Name()), childRel, segs, anchored, add); err != nil {
						return err
					}
				} else {
					add(childRel)
				}
			}
			return nil
		}
	}

	entries, err := os.ReadDir(physDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		childRel := joinVirtual(vrel, e.Name())
		if v.ignored(childRel, e.IsDir()) {
			continue
		}
		childPhys := filepath.Join(physDir, e.Name())

		if segs[0] == "**" {
			if e.IsDir() {
				if err := v.globWalk(childPhys, childRel, segs, anchored, add); err != nil {
					return err
				}
				if err := v.globWalk(childPhys, childRel, segs[1:], anchored, add); err != nil {
					return err
				}
			}
			continue
		}

		if !globMatch(segs[0], e.Name()) {
			continue
		}
		if len(segs) == 1 {
			add(childRel)
			continue
		}
		if e.IsDir() {
			if err := v.globWalk(childPhys, childRel, segs[1:], anchored, add); err != nil {
				return err
			}
		}
	}

	if !anchored && len(segs) == 1 && segs[0] != "**" {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			childRel := joinVirtual(vrel, e.Name())
			if v.ignored(childRel, true) {
				continue
			}
			if err := v.globWalk(filepath.Join(physDir, e.Name()), childRel, segs, anchored, add); err != nil {
				return err
			}
		}
	}
	return nil
}

type GrepMode string

const (
	GrepModeContent          = GrepMode("content")
	GrepModeFilesWithMatches = GrepMode("files_with_matches")
	GrepModeCount            = GrepMode("count")
)

type GrepResult struct {
	Path  string
	Line  int
	Text  string
	Count int
}

var errGrepLimit = errors.New("vfs: grep result limit reached")

func (v *VFS) Grep(pattern, path string, include []string, mode GrepMode, headLimit int) ([]GrepResult, error) {
	if mode == "" {
		mode = GrepModeContent
	}
	switch mode {
	case GrepModeContent, GrepModeFilesWithMatches, GrepModeCount:
	default:
		return nil, ErrInvalidGrepMode
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	if path == "" {
		path = "."
	}
	phys, rel, isDir, err := v.resolveChecked(path)
	if err != nil {
		return nil, err
	}
	if !isDir {
		return nil, ErrNotFound
	}

	inc := newIncludeMatcher(include)
	var out []GrepResult
	seen := make(map[string]bool)
	if err := v.grepWalk(phys, rel, re, inc, mode, headLimit, &out, seen); err != nil && !errors.Is(err, errGrepLimit) {
		return nil, err
	}
	return out, nil
}

func (v *VFS) grepWalk(physDir, vrel string, re *regexp.Regexp, inc *includeMatcher, mode GrepMode, headLimit int, out *[]GrepResult, seen map[string]bool) error {
	entries, err := os.ReadDir(physDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		childRel := joinVirtual(vrel, e.Name())
		if v.ignored(childRel, e.IsDir()) {
			continue
		}
		if e.IsDir() {
			if err := v.grepWalk(filepath.Join(physDir, e.Name()), childRel, re, inc, mode, headLimit, out, seen); err != nil {
				return err
			}
			continue
		}
		if !inc.match(childRel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(physDir, e.Name()))
		if err != nil || isBinary(data) {
			continue
		}

		switch mode {
		case GrepModeFilesWithMatches:
			if re.Match(data) && !seen[childRel] {
				seen[childRel] = true
				*out = append(*out, GrepResult{Path: childRel})
			}
		case GrepModeCount:
			if n := countLines(data, re); n > 0 {
				*out = append(*out, GrepResult{Path: childRel, Count: n})
			}
		default:
			for i, line := range strings.Split(string(data), "\n") {
				if re.MatchString(line) {
					*out = append(*out, GrepResult{Path: childRel, Line: i + 1, Text: line})
				}
			}
		}

		if headLimit > 0 && len(*out) >= headLimit {
			return errGrepLimit
		}
	}
	return nil
}

func countLines(data []byte, re *regexp.Regexp) int {
	n := 0
	for _, line := range strings.Split(string(data), "\n") {
		if re.MatchString(line) {
			n++
		}
	}
	return n
}

type includeMatcher struct {
	pats []pattern
}

func newIncludeMatcher(includes []string) *includeMatcher {
	m := &includeMatcher{}
	for _, inc := range includes {
		for _, expanded := range expandBraces(inc) {
			if p, ok := parsePattern(expanded); ok {
				m.pats = append(m.pats, p)
			}
		}
	}
	return m
}

func (m *includeMatcher) match(vpath string) bool {
	if len(m.pats) == 0 {
		return true
	}
	for _, p := range m.pats {
		if matches(p, vpath, false) {
			return true
		}
	}
	return false
}

func expandBraces(pattern string) []string {
	open := strings.IndexByte(pattern, '{')
	if open < 0 {
		return []string{pattern}
	}
	close := strings.IndexByte(pattern[open:], '}')
	if close < 0 {
		return []string{pattern}
	}
	close += open
	var out []string
	for _, alt := range strings.Split(pattern[open+1:close], ",") {
		out = append(out, expandBraces(pattern[:open]+alt+pattern[close+1:])...)
	}
	return out
}

func isBinary(data []byte) bool {
	limit := len(data)
	if limit > 8192 {
		limit = 8192
	}
	return bytes.IndexByte(data[:limit], 0) >= 0
}

func joinVirtual(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}
