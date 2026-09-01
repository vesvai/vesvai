package cache

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestSQLiteCache(t *testing.T) *SQLiteCache {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	c, err := newSQLiteCache(db)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestSQLiteCacheSetGet(t *testing.T) {
	c := newTestSQLiteCache(t)

	if err := c.Set("k1", []byte("value")); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("k1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "value" {
		t.Fatalf("got %q, want %q", got, "value")
	}
}

func TestSQLiteCacheSetUpdates(t *testing.T) {
	c := newTestSQLiteCache(t)

	if err := c.Set("k", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("k", []byte("b")); err != nil {
		t.Fatal(err)
	}

	got, err := c.Get("k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "b" {
		t.Fatalf("got %q, want %q", got, "b")
	}
}

func TestSQLiteCacheGetMissing(t *testing.T) {
	c := newTestSQLiteCache(t)

	if _, err := c.Get("missing"); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestSQLiteCacheDelete(t *testing.T) {
	c := newTestSQLiteCache(t)

	if err := c.Set("k", []byte("v")); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete("k"); err != nil {
		t.Fatal(err)
	}

	if _, err := c.Get("k"); err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestSQLiteCacheClear(t *testing.T) {
	c := newTestSQLiteCache(t)

	if err := c.Set("k1", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := c.Set("k2", []byte("b")); err != nil {
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

func TestSQLiteCacheClose(t *testing.T) {
	c := newTestSQLiteCache(t)

	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := c.Set("k", []byte("v")); err == nil {
		t.Fatal("expected error setting after close")
	}
}
