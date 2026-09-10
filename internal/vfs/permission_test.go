package vfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTestFS(t *testing.T) *VFS {
	t.Helper()
	root := t.TempDir()
	fs, err := New(root, Options{})
	if err != nil {
		t.Fatalf("mount vfs: %v", err)
	}
	return fs
}

func TestOutOfBoundsError(t *testing.T) {
	fs := newTestFS(t)

	_, err := fs.Resolve("/etc/passwd")
	var oob *OutOfBoundsError
	if !errors.As(err, &oob) {
		t.Fatalf("expected *OutOfBoundsError, got %T: %v", err, err)
	}
	if oob.Path == "" {
		t.Fatal("expected the offending path in the error")
	}
	if !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("errors.Is(err, ErrOutOfBounds) = false")
	}
}

func TestWithUnrestricted(t *testing.T) {
	fs := newTestFS(t)

	ctx := WithUnrestricted(context.Background())
	path, err := fs.ResolveCtx(ctx, "/etc/passwd")
	if err != nil {
		t.Fatalf("ResolveCtx with unrestricted ctx: %v", err)
	}
	if path != "/etc/passwd" {
		t.Fatalf("path = %q, want /etc/passwd", path)
	}
}

func TestWithPermittedPath(t *testing.T) {
	fs := newTestFS(t)
	outside := filepath.Join(filepath.Dir(fs.Root()), "outside.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := fs.ResolveCtx(context.Background(), outside)
	var oob *OutOfBoundsError
	if !errors.As(err, &oob) {
		t.Fatalf("expected out of bounds, got %v", err)
	}

	ctx := WithPermittedPath(context.Background(), oob.Path)
	path, err := fs.ResolveCtx(ctx, outside)
	if err != nil {
		t.Fatalf("ResolveCtx with permitted path: %v", err)
	}
	if path != filepath.Clean(outside) {
		t.Fatalf("path = %q, want %q", path, outside)
	}
}

func TestReadWithPermission(t *testing.T) {
	fs := newTestFS(t)
	outside := filepath.Join(filepath.Dir(fs.Root()), "secret.txt")
	if err := os.WriteFile(outside, []byte("top secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := fs.Read(outside); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("strict read should fail, got %v", err)
	}

	ctx := WithUnrestricted(context.Background())
	content, err := fs.ReadCtx(ctx, outside)
	if err != nil {
		t.Fatalf("ReadCtx with unrestricted ctx: %v", err)
	}
	if content == "" {
		t.Fatal("expected file content")
	}
}

func TestIgnoreStillEnforcedWithPermission(t *testing.T) {
	fs := newTestFS(t)
	if err := os.WriteFile(filepath.Join(fs.Root(), ".gitignore"), []byte("ignored.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fs.Root(), "ignored.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := WithUnrestricted(context.Background())
	if _, err := fs.ReadCtx(ctx, "ignored.txt"); !errors.Is(err, ErrIgnored) {
		t.Fatalf("ignore protection must stay enforced, got %v", err)
	}
}

func TestIgnoreDoesNotApplyOutsideWorkspace(t *testing.T) {
	fs := newTestFS(t)
	parent := filepath.Dir(fs.Root())

	if err := os.WriteFile(filepath.Join(fs.Root(), ".gitignore"), []byte("*.secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".gitignore"), []byte("*.secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "notes.secret")
	if err := os.WriteFile(outside, []byte("data\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := WithUnrestricted(context.Background())
	content, err := fs.ReadCtx(ctx, outside)
	if err != nil {
		t.Fatalf("permitted escape must bypass ignore rules outside the workspace, got %v", err)
	}
	if content == "" {
		t.Fatal("expected file content")
	}

	if _, err := fs.WriteCtx(ctx, filepath.Join(parent, "out.secret"), []byte("x")); err != nil {
		t.Fatalf("permitted write outside workspace must bypass ignore rules, got %v", err)
	}
}

func TestWriteWithPermission(t *testing.T) {
	fs := newTestFS(t)
	outsideDir := filepath.Join(filepath.Dir(fs.Root()), "writedir")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(outsideDir, "new.txt")

	if _, err := fs.Write(outside, []byte("data")); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("strict write should fail, got %v", err)
	}

	ctx := WithUnrestricted(context.Background())
	if _, err := fs.WriteCtx(ctx, outside, []byte("data")); err != nil {
		t.Fatalf("WriteCtx with unrestricted ctx: %v", err)
	}
}
