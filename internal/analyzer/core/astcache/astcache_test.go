package astcache

import (
	"go/token"
	"testing"
)

func TestParseAndCache(t *testing.T) {
	t.Parallel()

	c := New()
	src := []byte("package main\n\nfunc main() {}\n")

	f, err := c.Parse("test.go", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if f == nil {
		t.Fatal("parsed file is nil")
	}
	if f.Name.Name != "main" {
		t.Errorf("package name = %q, want main", f.Name.Name)
	}

	cached := c.Get("test.go")
	if cached == nil {
		t.Fatal("Get returned nil after Parse")
	}
	if cached != f {
		t.Error("Get returned a different pointer than Parse")
	}
}

func TestParseErrorCached(t *testing.T) {
	t.Parallel()

	c := New()
	src := []byte("package main\n\nfunc broken(\n")

	_, err := c.Parse("broken.go", src)
	if err == nil {
		t.Fatal("expected parse error for broken code")
	}

	// A second parse of the same broken file should return the same error
	// without re-parsing (cached).
	_, err2 := c.Parse("broken.go", src)
	if err2 == nil {
		t.Fatal("expected cached parse error on second call")
	}
}

func TestFileSetShared(t *testing.T) {
	t.Parallel()

	c := New()
	_, _ = c.Parse("a.go", []byte("package a\n"))
	_, _ = c.Parse("b.go", []byte("package b\n"))

	fs := c.FileSet()
	if fs == nil {
		t.Fatal("FileSet is nil")
	}
	count := 0
	fs.Iterate(func(*token.File) bool {
		count++
		return true
	})
	if count == 0 {
		t.Fatal("FileSet has no files")
	}
}
