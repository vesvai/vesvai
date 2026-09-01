package cache

import (
	"testing"
)

func newTestJSONCache(t *testing.T) *JSONCache {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	c, err := NewJSONCache()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestJSONCacheSetGet(t *testing.T) {
	c := newTestJSONCache(t)

	if err := c.Set("k1", []byte(`"v1"`)); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("k1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `"v1"` {
		t.Fatalf("got %q, want %q", got, `"v1"`)
	}
}

func TestJSONCacheGetMissing(t *testing.T) {
	c := newTestJSONCache(t)

	if _, err := c.Get("missing"); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestJSONCacheDelete(t *testing.T) {
	c := newTestJSONCache(t)

	if err := c.Set("k", []byte(`1`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete("k"); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Get("k"); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestJSONCacheSetUpdates(t *testing.T) {
	c := newTestJSONCache(t)

	if err := c.Set("k", []byte(`"a"`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("k", []byte(`"b"`)); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `"b"` {
		t.Fatalf("got %q, want %q", got, `"b"`)
	}
}

func TestJSONCacheClear(t *testing.T) {
	c := newTestJSONCache(t)

	if err := c.Set("k1", []byte(`"a"`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("k2", []byte(`"b"`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Get("k1"); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound after clear", err)
	}
	if _, err := c.Get("k2"); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound after clear", err)
	}
}

func TestJSONCacheReload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	c, err := NewJSONCache()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Set("k", []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewJSONCache()
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	got, err := reopened.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"a":1}` {
		t.Fatalf("got %q, want %q", got, `{"a":1}`)
	}
}
