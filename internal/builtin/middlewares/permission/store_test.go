package permission

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreHashAndKey(t *testing.T) {
	s := &Store{}
	h1 := s.Hash(`{"filePath": "a.txt"}`)
	h2 := s.Hash(`{"filePath": "a.txt"}`)
	h3 := s.Hash(`{"filePath": "b.txt"}`)
	if h1 != h2 {
		t.Fatal("same args must hash identically")
	}
	if h1 == h3 {
		t.Fatal("different args must hash differently")
	}
	if len(h1) != 64 {
		t.Fatalf("hash length = %d, want 64", len(h1))
	}
	if k := s.Key("read", `{"filePath": "a.txt"}`); k != "read:"+h1 {
		t.Fatalf("key = %q", k)
	}
}

func TestStoreHashCanonical(t *testing.T) {
	s := &Store{}
	cases := [][]string{
		{`{"filePath": "a.txt"}`, `{"filePath":"a.txt"}`, `{"filePath": "a.txt" }`},
		{`{"offset":1,"filePath":"a.txt"}`, `{"filePath":"a.txt","offset":1}`},
		{"{\n  \"filePath\": \"a.txt\"\n}", `{"filePath":"a.txt"}`},
		{`{"path":"x","nested":{"b":1,"a":2}}`, `{"path":"x","nested":{"a":2,"b":1}}`},
	}
	for i, group := range cases {
		base := s.Hash(group[0])
		for _, v := range group[1:] {
			if got := s.Hash(v); got != base {
				t.Errorf("case %d: hash(%q) = %s, want %s", i, v, got, base)
			}
		}
	}

	if s.Hash(`{"filePath":"a.txt","offset":1}`) == s.Hash(`{"filePath":"a.txt"}`) {
		t.Error("different parameters must hash differently")
	}
	if s.Hash(`{"offset":1}`) != s.Hash(`{"offset": 1.0}`) {
		t.Error("numeric normalization expected")
	}
}

func TestStoreHashNonJSONFallback(t *testing.T) {
	s := &Store{}
	a := `not json at all`
	b := `  ` + a + `  `
	if s.Hash(a) != s.Hash(b) {
		t.Fatal("non-JSON args must hash the trimmed raw string")
	}
	if s.Hash(a) == s.Hash(a+"x") {
		t.Fatal("different non-JSON args must differ")
	}
}

func TestStoreAllowRejectRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := &Store{path: filepath.Join(dir, "permissions.json"), data: &storeData{
		Allowed:  map[string]*allowedEntry{},
		Rejected: map[string]*rejectedEntry{},
	}}

	s.Allow("read", `{"filePath": "a.txt"}`)
	key := s.Key("read", `{"filePath": "a.txt"}`)
	if !s.IsAllowed(key) {
		t.Fatal("expected allowed after Allow")
	}
	if _, ok := s.RejectionReason(key); ok {
		t.Fatal("unexpected rejection")
	}

	s.Reject("write", `{"filePath": "b.txt"}`, "user said no")
	rejKey := s.Key("write", `{"filePath": "b.txt"}`)
	reason, ok := s.RejectionReason(rejKey)
	if !ok || reason != "user said no" {
		t.Fatalf("rejection reason = %q, %v", reason, ok)
	}

	if s.IsAllowed(s.Key("read", `{"filePath": "other.txt"}`)) {
		t.Fatal("different args must not be allowed")
	}
}

func TestStorePersistsToDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "permissions.json")
	s := &Store{path: path, data: &storeData{
		Allowed:  map[string]*allowedEntry{},
		Rejected: map[string]*rejectedEntry{},
	}}
	s.Allow("read", `{"filePath": "a.txt"}`)

	reloaded := &Store{path: path}
	reloaded.load()
	if !reloaded.IsAllowed(reloaded.Key("read", `{"filePath": "a.txt"}`)) {
		t.Fatal("allowed entry must survive reload")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("permissions.json must be written")
	}
}

func TestStoreLoadMissingFile(t *testing.T) {
	s := NewStore()
	s.path = filepath.Join(t.TempDir(), "does-not-exist.json")
	s.load()
	if s.data == nil || s.data.Allowed == nil || s.data.Rejected == nil {
		t.Fatal("store must initialize empty maps")
	}
}
