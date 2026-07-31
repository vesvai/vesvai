package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules/\n"), 0644)
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world\n"), 0644)
	os.WriteFile(filepath.Join(dir, "code.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "app.js"), []byte("console.log('hi');\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte("// dep\n"), 0644)
	return dir
}

func TestNewFileSystem(t *testing.T) {
	dir := setupTestDir(t)
	fs, err := New(Config{RootDir: dir, IgnoreDotfiles: true})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if fs.Root() != dir {
		t.Errorf("expected root %q, got %q", dir, fs.Root())
	}
}

func TestNewFileSystem_NotDir(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "file")
	defer f.Close()
	_, err := New(Config{RootDir: f.Name()})
	if err == nil {
		t.Fatal("expected error for non-directory root")
	}
}

func TestRead(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	content, err := fs.Read(context.Background(), "hello.txt")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if content != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", content)
	}
}

func TestRead_Binary(t *testing.T) {
	dir := setupTestDir(t)
	binPath := filepath.Join(dir, "image.png")
	os.WriteFile(binPath, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}, 0644)

	fs, _ := New(Config{RootDir: dir})
	content, err := fs.Read(context.Background(), "image.png")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content for binary file")
	}
	if !contains(content, "Binary file") {
		t.Errorf("expected binary file message, got %q", content)
	}
}

func TestRead_IgnoredFile(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0644)

	fs, _ := New(Config{RootDir: dir, IgnoreDotfiles: true})
	_, err := fs.Read(context.Background(), ".hidden")
	if err == nil {
		t.Fatal("expected error for dotfile with IgnoreDotfiles=true")
	}
}

func TestRead_IgnoredFile_Allowed(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("secret"), 0644)

	fs, _ := New(Config{RootDir: dir, IgnoreDotfiles: false})
	content, err := fs.Read(context.Background(), ".hidden")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if content != "secret" {
		t.Errorf("expected 'secret', got %q", content)
	}
}

func TestReadRange(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, "lines.txt"), []byte("a\nb\nc\nd\ne\n"), 0644)

	fs, _ := New(Config{RootDir: dir})
	content, err := fs.ReadRange(context.Background(), "lines.txt", 2, 4)
	if err != nil {
		t.Fatalf("ReadRange failed: %v", err)
	}
	expected := "2: b\n3: c\n4: d\n"
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

func TestWrite(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	err := fs.Write(context.Background(), "new.txt", []byte("new content"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "new.txt"))
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got %q", string(data))
	}
}

func TestWrite_CreatesDirs(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	err := fs.Write(context.Background(), "deep/nested/file.txt", []byte("nested"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "deep/nested/file.txt"))
	if string(data) != "nested" {
		t.Errorf("expected 'nested', got %q", string(data))
	}
}

func TestWrite_EditConflict(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	fs.Read(context.Background(), "hello.txt")

	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("changed externally\n"), 0644)

	err := fs.Write(context.Background(), "hello.txt", []byte("new content"))
	if err == nil {
		t.Fatal("expected EditConflictError")
	}
	if _, ok := err.(*EditConflictError); !ok {
		t.Errorf("expected EditConflictError, got %T", err)
	}
}

func TestEdit(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("hello world\nfoo bar\n"), 0644)

	fs, _ := New(Config{RootDir: dir})
	err := fs.Edit(context.Background(), "edit.txt", "world", "universe", 1)
	if err != nil {
		t.Fatalf("Edit failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "edit.txt"))
	expected := "hello universe\nfoo bar\n"
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

func TestEdit_NotFound(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("hello world\n"), 0644)

	fs, _ := New(Config{RootDir: dir})
	err := fs.Edit(context.Background(), "edit.txt", "nonexistent", "new", 1)
	if err == nil {
		t.Fatal("expected error for non-matching old string")
	}
}

func TestEdit_Conflict(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("original\n"), 0644)

	fs, _ := New(Config{RootDir: dir})
	fs.Read(context.Background(), "edit.txt")

	os.WriteFile(filepath.Join(dir, "edit.txt"), []byte("external change\n"), 0644)

	err := fs.Edit(context.Background(), "edit.txt", "original", "new", 1)
	if err == nil {
		t.Fatal("expected EditConflictError")
	}
	if _, ok := err.(*EditConflictError); !ok {
		t.Errorf("expected EditConflictError, got %T", err)
	}
}

func TestList(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir, IgnoreDotfiles: true})

	entries, err := fs.List(context.Background(), ".")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	names := make(map[string]bool)
	for _, e := range entries {
		names[e.Name] = true
	}

	if !names["hello.txt"] {
		t.Error("expected hello.txt in listing")
	}
	if !names["src"] {
		t.Error("expected src in listing")
	}
	if names["node_modules"] {
		t.Error("node_modules should be ignored by .gitignore")
	}
}

func TestList_DotfilesFiltered(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=x"), 0644)

	fs, _ := New(Config{RootDir: dir, IgnoreDotfiles: true})
	entries, err := fs.List(context.Background(), ".")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	for _, e := range entries {
		if e.Name == ".env" {
			t.Error(".env should be filtered by IgnoreDotfiles")
		}
	}
}

func TestGlob(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	matches, err := fs.Glob(context.Background(), "*.txt")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) != 1 || matches[0].Name != "hello.txt" {
		t.Errorf("expected [hello.txt], got %v", matches)
	}
}

func TestGlob_IgnoredExcluded(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	matches, err := fs.Glob(context.Background(), "node_modules/**/*")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("expected no matches in node_modules, got %v", matches)
	}
}

func TestWalk(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	var visited []string
	err := fs.Walk(context.Background(), func(relPath string, info FileInfo) error {
		visited = append(visited, relPath)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	for _, p := range visited {
		if contains(p, "node_modules") {
			t.Errorf("node_modules should not be visited, got %s", p)
		}
	}
}

func TestGrep(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	results, err := fs.Grep(context.Background(), "func", ".")
	if err != nil {
		t.Fatalf("Grep failed: %v", err)
	}

	found := false
	for _, r := range results {
		if r.Path == "code.go" && contains(r.Content, "func main()") {
			found = true
		}
	}
	if !found {
		t.Error("expected to find 'func main()' in code.go")
	}
}

func TestGrep_SkipsIgnored(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("vendor/\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "vendor"), 0755)
	os.WriteFile(filepath.Join(dir, "vendor", "lib.go"), []byte("package vendor\n"), 0644)

	results, _ := fs(t, dir).Grep(context.Background(), "package", ".")
	for _, r := range results {
		if contains(r.Path, "vendor") {
			t.Error("vendor should be ignored")
		}
	}
}

func TestIsIgnored(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir, IgnoreDotfiles: true})

	if !fs.IsIgnored("node_modules/pkg/index.js") {
		t.Error("node_modules should be ignored")
	}
	if fs.IsIgnored("hello.txt") {
		t.Error("hello.txt should not be ignored")
	}
	if !fs.IsIgnored(".hidden") {
		t.Error(".hidden should be ignored with IgnoreDotfiles=true")
	}
}

func TestResolve(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	resolved, err := fs.Resolve("hello.txt")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if resolved != filepath.Join(dir, "hello.txt") {
		t.Errorf("expected %s, got %s", filepath.Join(dir, "hello.txt"), resolved)
	}
}

func TestResolve_Traversal(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	_, err := fs.Resolve("../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestIgnoreGitignore(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\ntmp/\n"), 0644)
	os.WriteFile(filepath.Join(dir, "debug.log"), []byte("log data"), 0644)
	os.MkdirAll(filepath.Join(dir, "tmp"), 0755)
	os.WriteFile(filepath.Join(dir, "tmp", "cache.dat"), []byte("cache"), 0644)

	fs, _ := New(Config{RootDir: dir})

	if !fs.IsIgnored("debug.log") {
		t.Error("*.log should be ignored by .gitignore")
	}
	if !fs.IsIgnored("tmp/cache.dat") {
		t.Error("tmp/ should be ignored by .gitignore")
	}
}

func TestIgnoreVesvaignore(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".vesvaignore"), []byte("secrets/\n*.key\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "secrets"), 0755)
	os.WriteFile(filepath.Join(dir, "secrets", "api.key"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(dir, "server.key"), []byte("key data"), 0644)

	fs, _ := New(Config{RootDir: dir})

	if !fs.IsIgnored("secrets/api.key") {
		t.Error("secrets/ should be ignored by .vesvaignore")
	}
	if !fs.IsIgnored("server.key") {
		t.Error("*.key should be ignored by .vesvaignore")
	}
}

func TestIgnoreNegation(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.txt\n!important.txt\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignored"), 0644)
	os.WriteFile(filepath.Join(dir, "important.txt"), []byte("not ignored"), 0644)

	fs, _ := New(Config{RootDir: dir})

	if fs.IsIgnored("important.txt") {
		t.Error("important.txt should NOT be ignored due to negation in .gitignore")
	}
	if !fs.IsIgnored("readme.txt") {
		t.Error("readme.txt should be ignored")
	}
}

func TestIgnoreDoubleStar(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("**/temp\n"), 0644)
	os.MkdirAll(filepath.Join(dir, "a", "b", "temp"), 0755)
	os.WriteFile(filepath.Join(dir, "a", "b", "temp", "file.txt"), []byte("x"), 0644)

	fs, _ := New(Config{RootDir: dir})

	if !fs.IsIgnored("a/b/temp/file.txt") {
		t.Error("**/temp should match temp at any depth")
	}
}

func fs(t *testing.T, dir string) *FileSystem {
	t.Helper()
	fs, err := New(Config{RootDir: dir, IgnoreDotfiles: true})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return fs
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGlob_RecursiveStarGo(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	matches, err := fs.Glob(context.Background(), "*.go")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(matches) != 1 || matches[0].Name != "code.go" {
		t.Errorf("expected [code.go], got %v", matches)
	}
}

func TestGlob_DeepRecursive(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})
	os.MkdirAll(filepath.Join(dir, "cmd", "deep", "nested"), 0755)
	os.WriteFile(filepath.Join(dir, "cmd", "deep", "nested", "main.go"), []byte("package main\n"), 0644)

	matches, err := fs.Glob(context.Background(), "**/*.go")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 go files (root + nested), got %v", matches)
	}
}

func TestGlob_PatternWithBaseDir(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})
	os.MkdirAll(filepath.Join(dir, "src", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "src", "deep", "util.js"), []byte("x\n"), 0644)

	matches, err := fs.Glob(context.Background(), "src/**/*.js")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("expected 2 js files under src, got %v", matches)
	}
}

func TestGlob_ZeroOrMoreSegments(t *testing.T) {
	dir := setupTestDir(t)
	fs, _ := New(Config{RootDir: dir})

	matches, err := fs.Glob(context.Background(), "**/*.txt")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}
	if len(matches) != 1 || matches[0].Name != "hello.txt" {
		t.Errorf("expected [hello.txt], got %v", matches)
	}
}

func TestMatchGlobSegs(t *testing.T) {
	cases := []struct {
		pat, name []string
		want      bool
	}{
		{[]string{"**", "*.go"}, []string{"main.go"}, true},
		{[]string{"**", "*.go"}, []string{"a", "b", "main.go"}, true},
		{[]string{"**", "*.go"}, []string{"a", "main.txt"}, false},
		{[]string{"src", "**", "*.go"}, []string{"src", "main.go"}, true},
		{[]string{"src", "**", "*.go"}, []string{"src", "deep", "main.go"}, true},
		{[]string{"src", "**", "*.go"}, []string{"other", "main.go"}, false},
		{[]string{"*.go"}, []string{"main.go"}, true},
		{[]string{"*.go"}, []string{"a", "main.go"}, false},
	}
	for _, c := range cases {
		if got := matchGlobSegs(c.pat, c.name); got != c.want {
			t.Errorf("matchGlobSegs(%v, %v) = %v, want %v", c.pat, c.name, got, c.want)
		}
	}
}

func TestGlobBaseDir(t *testing.T) {
	cases := []struct{ pat, want string }{
		{"**/*.go", ""},
		{"src/**/*.go", "src"},
		{"a/b/*.txt", "a/b"},
		{"*.go", ""},
		{"cmd/deep/nested/main.go", ""},
	}
	for _, c := range cases {
		if got := globBaseDir(c.pat); got != c.want {
			t.Errorf("globBaseDir(%q) = %q, want %q", c.pat, got, c.want)
		}
	}
}
