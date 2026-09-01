package vfs

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func buildTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestList(t *testing.T) {
	root := buildTree(t, map[string]string{
		"src/a.go":       "package a",
		"src/b.go":       "package b",
		"node_modules/x": "ignored",
		"README.md":      "# readme",
		".git/config":    "dummy",
	})
	writeIgnoreFiles(t, root, "node_modules/\n", "")

	fs := newTestVFS(t, root)

	entries, err := fs.List(".")
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries.Entries))
	for i, e := range entries.Entries {
		names[i] = e.Name
	}
	if len(names) != 3 || names[0] != ".gitignore" || names[1] != "README.md" || names[2] != "src" {
		t.Fatalf("unexpected root listing: %v", names)
	}
	if entries.FileCount != 2 || entries.DirCount != 1 {
		t.Fatalf("counts mismatch: files=%d dirs=%d", entries.FileCount, entries.DirCount)
	}

	src, err := fs.List("src")
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Entries) != 2 || src.Entries[0].Name != "a.go" || src.Entries[1].Name != "b.go" {
		t.Fatalf("unexpected src listing: %+v", src.Entries)
	}
	if src.FileCount != 2 || src.DirCount != 0 {
		t.Fatalf("src counts mismatch: files=%d dirs=%d", src.FileCount, src.DirCount)
	}

	if _, err := fs.List("missing"); !errorsIsNotFound(err) {
		t.Fatalf("want not found, got %v", err)
	}
	if _, err := fs.List("README.md"); !errorsIsNotFound(err) {
		t.Fatalf("listing a file should fail, got %v", err)
	}
}

func errorsIsNotFound(err error) bool {
	return err != nil && err.Error() == "vfs: file not found"
}

func errorsIsOutOfBounds(err error) bool {
	return err != nil && err.Error() == "vfs: path escapes the workspace root"
}

func TestGlob(t *testing.T) {
	root := buildTree(t, map[string]string{
		"src/a.go":          "package a",
		"src/b.txt":         "text",
		"src/nested/c.go":   "package c",
		"cmd/main.go":       "package main",
		"node_modules/x.js": "ignored",
		"README.md":         "# readme",
	})
	writeIgnoreFiles(t, root, "node_modules/\n", "")

	fs := newTestVFS(t, root)

	tests := []struct {
		pattern string
		path    string
		want    []string
	}{
		{"*.go", ".", []string{"cmd/main.go", "src/a.go", "src/nested/c.go"}},
		{"**/*.go", ".", []string{"cmd/main.go", "src/a.go", "src/nested/c.go"}},
		{"src/*.go", ".", []string{"src/a.go"}},
		{"**/*.txt", ".", []string{"src/b.txt"}},
		{"README.md", ".", []string{"README.md"}},
		{"src/nested/*", ".", []string{"src/nested/c.go"}},
		{"**/*", ".", []string{".gitignore", "README.md", "cmd", "cmd/main.go", "src", "src/a.go", "src/b.txt", "src/nested", "src/nested/c.go"}},
		{"*.go", "src", []string{"src/a.go", "src/nested/c.go"}},
		{"*.go", "src/nested", []string{"src/nested/c.go"}},
		{"b.txt", "src", []string{"src/b.txt"}},
	}
	for _, tt := range tests {
		got, err := fs.Glob(tt.pattern, tt.path)
		if err != nil {
			t.Fatalf("Glob(%q, %q): %v", tt.pattern, tt.path, err)
		}
		sorted := append([]string(nil), got...)
		sort.Strings(sorted)
		if len(sorted) != len(tt.want) {
			t.Fatalf("Glob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
		for i := range sorted {
			if sorted[i] != tt.want[i] {
				t.Fatalf("Glob(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		}
	}

	if _, err := fs.Glob("../escape", "."); !errorsIsOutOfBounds(err) {
		t.Fatalf("traversal glob = %v, want ErrOutOfBounds", err)
	}
	if _, err := fs.Glob("/etc/passwd", "."); !errorsIsOutOfBounds(err) {
		t.Fatalf("absolute glob = %v, want ErrOutOfBounds", err)
	}
}

func TestGlobSortedByModTime(t *testing.T) {
	root := buildTree(t, map[string]string{
		"old.go": "old",
		"mid.go": "mid",
		"new.go": "new",
	})
	fs := newTestVFS(t, root)

	old := filepath.Join(root, "old.go")
	mid := filepath.Join(root, "mid.go")
	new := filepath.Join(root, "new.go")
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(old, base, base); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(mid, base.Add(time.Hour), base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(new, base.Add(2*time.Hour), base.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := fs.Glob("*.go", ".")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"new.go", "mid.go", "old.go"}
	if len(got) != len(want) {
		t.Fatalf("Glob = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("Glob = %v, want %v (most recent first)", got, want)
		}
	}
}

func TestGrep(t *testing.T) {
	root := buildTree(t, map[string]string{
		"src/a.go":          "package a\n// foo function\nfunc Foo() {}\n",
		"src/b.txt":         "foo bar\nfoo baz\n",
		"secret.key":        "foo foo\n",
		"binary.dat":        "\x00\x01\x02foo\n",
		"node_modules/x.js": "foo\n",
	})
	writeIgnoreFiles(t, root, "secret.key\nnode_modules/\n", "")

	fs := newTestVFS(t, root)

	matches, err := fs.Grep("foo", ".", nil, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fatalf("matches = %+v", matches)
	}
	if matches[0].Path != "src/a.go" || matches[0].Line != 2 || matches[0].Text != "// foo function" {
		t.Fatalf("unexpected first match: %+v", matches[0])
	}
	if matches[1].Path != "src/b.txt" || matches[1].Line != 1 {
		t.Fatalf("unexpected second match: %+v", matches[1])
	}
	if matches[2].Path != "src/b.txt" || matches[2].Line != 2 {
		t.Fatalf("unexpected third match: %+v", matches[2])
	}
}

func TestGrepInclude(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a.go":     "foo\n",
		"b.ts":     "foo\n",
		"c.tsx":    "foo\n",
		"d.txt":    "foo\n",
		"src/e.go": "foo\n",
	})
	fs := newTestVFS(t, root)

	matches, err := fs.Grep("foo", ".", []string{"*.go", "*.{ts,tsx}"}, GrepModeFilesWithMatches, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, len(matches))
	for i, m := range matches {
		got[i] = m.Path
	}
	want := []string{"a.go", "b.ts", "c.tsx", "src/e.go"}
	if len(got) != len(want) {
		t.Fatalf("include filter = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("include filter = %v, want %v", got, want)
		}
	}
}

func TestGrepModes(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a.txt": "foo\nno\nfoo\n",
		"b.txt": "foo\n",
	})
	fs := newTestVFS(t, root)

	files, err := fs.Grep("foo", ".", nil, GrepModeFilesWithMatches, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0].Path != "a.txt" || files[1].Path != "b.txt" {
		t.Fatalf("files_with_matches = %+v", files)
	}

	counts, err := fs.Grep("foo", ".", nil, GrepModeCount, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 2 || counts[0].Path != "a.txt" || counts[0].Count != 2 || counts[1].Count != 1 {
		t.Fatalf("count = %+v", counts)
	}

	if _, err := fs.Grep("foo", ".", nil, "bogus", 0); !errorsIsInvalidMode(err) {
		t.Fatalf("invalid mode = %v, want ErrInvalidGrepMode", err)
	}
}

func TestGrepHeadLimit(t *testing.T) {
	root := buildTree(t, map[string]string{
		"a.go": "foo\n",
		"b.go": "foo\n",
		"c.go": "foo\n",
	})
	fs := newTestVFS(t, root)

	matches, err := fs.Grep("foo", ".", nil, GrepModeContent, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("head_limit = %d, want 2", len(matches))
	}

	matches, err = fs.Grep("foo", ".", nil, GrepModeFilesWithMatches, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("head_limit files = %d, want 1", len(matches))
	}
}

func errorsIsInvalidMode(err error) bool {
	return err != nil && err.Error() == "vfs: invalid grep output mode"
}
