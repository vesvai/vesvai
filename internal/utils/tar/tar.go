package tar

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Entry struct {
	Name string
	Mode int64
	Body []byte
}

func WriteGz(dst string, entries []Entry) error {
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("tar: create %s: %w", dst, err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		name := filepath.ToSlash(filepath.Clean(filepath.FromSlash(e.Name)))
		if err := validateName(name); err != nil {
			return err
		}
		mode := e.Mode
		if mode == 0 {
			mode = 0o644
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     mode,
			Size:     int64(len(e.Body)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			return fmt.Errorf("tar: write header %s: %w", name, err)
		}
		if _, err := tw.Write(e.Body); err != nil {
			return fmt.Errorf("tar: write body %s: %w", name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("tar: close archive writer: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("tar: close gzip writer: %w", err)
	}
	return nil
}

func ListGz(src string) ([]string, error) {
	tr, cleanup, err := openReader(src)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var names []string
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: read %s: %w", src, err)
		}
		if h.Typeflag == tar.TypeReg {
			names = append(names, h.Name)
		}
	}
	return names, nil
}

func ExtractGz(src, dst string) ([]string, error) {
	tr, cleanup, err := openReader(src)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	if err := os.MkdirAll(dst, 0o755); err != nil {
		return nil, fmt.Errorf("tar: create dir %s: %w", dst, err)
	}

	var out []string
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: read %s: %w", src, err)
		}

		name := filepath.ToSlash(filepath.Clean(filepath.FromSlash(h.Name)))
		if err := validateName(name); err != nil {
			return nil, err
		}
		dest := filepath.Join(dst, filepath.FromSlash(name))
		if !within(dst, dest) {
			return nil, fmt.Errorf("tar: entry %q escapes extraction dir", h.Name)
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return nil, fmt.Errorf("tar: mkdir %s: %w", dest, err)
			}
		case tar.TypeReg:
			mode := os.FileMode(h.Mode & 0o777)
			if mode == 0 {
				mode = 0o644
			}
			if err := writeFile(dest, tr, mode); err != nil {
				return nil, fmt.Errorf("tar: extract %s: %w", dest, err)
			}
			out = append(out, dest)
		default:
		}
	}
	return out, nil
}

func openReader(src string) (*tar.Reader, func(), error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, nil, fmt.Errorf("tar: open %s: %w", src, err)
	}
	cleanup := func() { _ = f.Close() }

	head := make([]byte, 2)
	n, err := io.ReadFull(f, head)
	if err != nil || n < 2 {
		return nil, cleanup, fmt.Errorf("tar: read %s: %w", src, io.ErrUnexpectedEOF)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, cleanup, fmt.Errorf("tar: seek %s: %w", src, err)
	}

	if head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(f)
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("tar: gzip %s: %w", src, err)
		}
		cleanup = func() { _ = gz.Close(); _ = f.Close() }
		return tar.NewReader(gz), cleanup, nil
	}
	return tar.NewReader(f), cleanup, nil
}

func writeFile(dest string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chmod(dest, mode)
}

func validateName(name string) error {
	if name == "" || name == "." || filepath.IsAbs(name) {
		return fmt.Errorf("tar: unsafe archive name %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return fmt.Errorf("tar: unsafe archive name %q", name)
		}
	}
	return nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
