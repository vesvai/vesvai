package tar

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func buildRawGz(t *testing.T, files map[string][]byte, extra ...*tar.Header) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range extra {
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestWriteAndListGz(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "a.tar.gz")
	err := WriteGz(dst, []Entry{
		{Name: "terraform-ls", Mode: 0o755, Body: []byte("bin1")},
		{Name: "bin/helper", Mode: 0o644, Body: []byte("bin2")},
	})
	if err != nil {
		t.Fatal(err)
	}

	names, err := ListGz(dst)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"terraform-ls", "bin/helper"}
	if len(names) != len(want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("names = %v, want %v", names, want)
		}
	}
}

func TestWriteAndExtractGz(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.tar.gz")
	err := WriteGz(src, []Entry{
		{Name: "tools/terraform-ls", Mode: 0o755, Body: []byte("#!/bin/sh\necho hi\n")},
		{Name: "docs/readme.txt", Body: []byte("hello")},
	})
	if err != nil {
		t.Fatal(err)
	}

	dst := t.TempDir()
	extracted, err := ExtractGz(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	wantFiles := []string{
		filepath.Join(dst, "tools", "terraform-ls"),
		filepath.Join(dst, "docs", "readme.txt"),
	}
	if len(extracted) != 2 {
		t.Fatalf("extracted = %v, want %v", extracted, wantFiles)
	}
	for i, f := range extracted {
		if f != wantFiles[i] {
			t.Fatalf("extracted = %v, want %v", extracted, wantFiles)
		}
	}

	data, err := os.ReadFile(wantFiles[0])
	if err != nil || string(data) != "#!/bin/sh\necho hi\n" {
		t.Fatalf("content mismatch: %v %q", err, data)
	}
	fi, _ := os.Stat(wantFiles[0])
	if fi.Mode().Perm()&0o700 != 0o700 {
		t.Fatalf("mode = %v, want owner-writable+executable", fi.Mode().Perm())
	}

	fi2, _ := os.Stat(wantFiles[1])
	if fi2.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v, want 0644", fi2.Mode().Perm())
	}
}

func TestExtractGzNestedDirs(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := WriteGz(src, []Entry{
		{Name: "a/b/c/tool", Mode: 0o755, Body: []byte("x")},
	}); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if _, err := ExtractGz(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "a", "b", "c", "tool")); err != nil {
		t.Fatalf("nested file not extracted: %v", err)
	}
}

func TestExtractGzRejectsTraversal(t *testing.T) {
	dst := t.TempDir()
	outside := filepath.Join(filepath.Dir(dst), "evil.txt")
	_ = os.Remove(outside)

	src := filepath.Join(t.TempDir(), "malicious.tar.gz")
	os.WriteFile(src, buildRawGz(t, map[string][]byte{"../evil.txt": []byte("pwned")}), 0o644)

	if _, err := ExtractGz(src, dst); err == nil {
		t.Fatal("expected traversal entry to be rejected")
	}
	if _, err := os.Stat(outside); err == nil {
		t.Fatal("file was written outside the extraction dir")
	}
}

func TestExtractGzRejectsAbsolutePath(t *testing.T) {
	dst := t.TempDir()
	src := filepath.Join(t.TempDir(), "malicious.tar.gz")
	os.WriteFile(src, buildRawGz(t, map[string][]byte{"/tmp/evil-abs.txt": []byte("pwned")}), 0o644)

	if _, err := ExtractGz(src, dst); err == nil {
		t.Fatal("expected absolute entry to be rejected")
	}
	if _, err := os.Stat("/tmp/evil-abs.txt"); err == nil {
		t.Fatal("absolute path was written")
	}
}

func TestExtractGzSkipsSymlinks(t *testing.T) {
	dst := t.TempDir()
	src := filepath.Join(t.TempDir(), "sym.tar.gz")
	os.WriteFile(src, buildRawGz(t,
		map[string][]byte{"real.txt": []byte("data")},
		&tar.Header{Name: "link.txt", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"},
	), 0o644)

	extracted, err := ExtractGz(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(extracted) != 1 || filepath.Base(extracted[0]) != "real.txt" {
		t.Fatalf("extracted = %v", extracted)
	}
	if _, err := os.Stat(filepath.Join(dst, "link.txt")); err == nil {
		t.Fatal("symlink entry should be skipped")
	}
}

func TestExtractGzMissingArchive(t *testing.T) {
	if _, err := ExtractGz(filepath.Join(t.TempDir(), "nope.tar.gz"), t.TempDir()); err == nil {
		t.Fatal("expected error for missing archive")
	}
}

func TestExtractGzCorrupt(t *testing.T) {
	src := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	os.WriteFile(src, []byte("this is not a gzip archive at all"), 0o644)
	if _, err := ExtractGz(src, t.TempDir()); err == nil {
		t.Fatal("expected error for corrupt archive")
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("garbage-not-tar"))
	gz.Close()
	src2 := filepath.Join(t.TempDir(), "corrupt2.tar.gz")
	os.WriteFile(src2, buf.Bytes(), 0o644)
	if _, err := ExtractGz(src2, t.TempDir()); err == nil {
		t.Fatal("expected error for corrupt tar payload")
	}
}

func TestExtractGzEmptyArchive(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Close()
	src := filepath.Join(t.TempDir(), "empty.tar.gz")
	os.WriteFile(src, buf.Bytes(), 0o644)

	files, err := ExtractGz(src, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("files = %v, want none", files)
	}
}

func TestExtractGzPlainTar(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	body := []byte("plain tar binary")
	tw.WriteHeader(&tar.Header{Name: "tool", Mode: 0o755, Size: int64(len(body))})
	tw.Write(body)
	tw.Close()

	src := filepath.Join(t.TempDir(), "plain.tar")
	os.WriteFile(src, buf.Bytes(), 0o644)

	dst := t.TempDir()
	files, err := ExtractGz(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != filepath.Join(dst, "tool") {
		t.Fatalf("files = %v", files)
	}
}

func TestWriteGzRejectsUnsafeName(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "a.tar.gz")
	for _, name := range []string{"../evil", "/abs/evil", "", ".", "a/../../evil"} {
		if err := WriteGz(dst, []Entry{{Name: name, Body: []byte("x")}}); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestExtractGzModePreserved(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := WriteGz(src, []Entry{
		{Name: "run.sh", Mode: 0o755, Body: []byte("#!/bin/sh\n")},
		{Name: "data.txt", Mode: 0o600, Body: []byte("secret")},
	}); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if _, err := ExtractGz(src, dst); err != nil {
		t.Fatal(err)
	}

	fi, _ := os.Stat(filepath.Join(dst, "run.sh"))
	if fi.Mode().Perm() != 0o755 {
		t.Fatalf("run.sh mode = %v, want 0755", fi.Mode().Perm())
	}
	fi2, _ := os.Stat(filepath.Join(dst, "data.txt"))
	if fi2.Mode().Perm() != 0o600 {
		t.Fatalf("data.txt mode = %v, want 0600", fi2.Mode().Perm())
	}
}

func TestExtractGzUntrustedGzipBombGuard(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "empty.bin", Mode: 0o644, Size: 0})
	tw.Close()
	gz.Close()

	src := filepath.Join(t.TempDir(), "zero.tar.gz")
	os.WriteFile(src, buf.Bytes(), 0o644)

	dst := t.TempDir()
	files, err := ExtractGz(src, dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("files = %v", files)
	}
	fi, err := os.Stat(files[0])
	if err != nil || fi.Size() != 0 {
		t.Fatalf("empty file not extracted correctly: %v %v", fi, err)
	}
}

func TestListGzMissing(t *testing.T) {
	if _, err := ListGz(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteGzRoundTripThroughStdlib(t *testing.T) {
	src := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := WriteGz(src, []Entry{
		{Name: "tool", Mode: 0o755, Body: []byte("hello")},
	}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	h, err := tr.Next()
	if err != nil {
		t.Fatal(err)
	}
	if h.Name != "tool" || h.Mode != 0o755 {
		t.Fatalf("header = %+v", h)
	}
	data, _ := io.ReadAll(tr)
	if string(data) != "hello" {
		t.Fatalf("body = %q", data)
	}
}
