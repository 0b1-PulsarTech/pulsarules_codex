package arch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectIndex_CachesAcrossCalls(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "foo.go", "package foo\n")
	writeFile(t, tmp, "bar/baz.go", "package bar\n")

	idx1 := loadProjectIndex(tmp, "example.com/mod")
	if idx1 == nil || len(idx1.pkgMap) == 0 {
		t.Fatal("expected non-empty project index")
	}

	// Second call should return the cached index (same pointer).
	idx2 := loadProjectIndex(tmp, "example.com/mod")
	if idx1 != idx2 {
		t.Error("expected same cached index pointer")
	}
}

func TestLoadProjectIndex_InvalidatesOnFileCountChange(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "foo.go", "package foo\n")

	idx1 := loadProjectIndex(tmp, "example.com/mod")

	// Add a new file; the file count changes, so the cache should invalidate.
	writeFile(t, tmp, "bar.go", "package bar\n")
	idx2 := loadProjectIndex(tmp, "example.com/mod")
	if idx1 == idx2 {
		t.Error("expected a new index after file count change")
	}
}

func TestHashGoMod(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// No go.mod → empty hash.
	if h := hashGoMod(tmp); h != "" {
		t.Errorf("expected empty hash without go.mod, got %q", h)
	}

	// With go.mod → non-empty hash.
	if err := os.WriteFile(
		filepath.Join(tmp, "go.mod"),
		[]byte("module test\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	h1 := hashGoMod(tmp)
	if h1 == "" {
		t.Error("expected non-empty hash with go.mod")
	}

	// Same content → same hash.
	h2 := hashGoMod(tmp)
	if h1 != h2 {
		t.Error("expected same hash for same go.mod content")
	}

	// Different content → different hash.
	if err := os.WriteFile(
		filepath.Join(tmp, "go.mod"),
		[]byte("module other\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	h3 := hashGoMod(tmp)
	if h1 == h3 {
		t.Error("expected different hash after go.mod change")
	}
}

func TestCountGoFiles(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	writeFile(t, tmp, "a.go", "package foo\n")
	writeFile(t, tmp, "b_test.go", "package foo\n")
	writeFile(t, tmp, "sub/c.go", "package sub\n")

	count := countGoFiles(tmp)
	if count != 2 {
		t.Errorf("expected 2 non-test .go files, got %d", count)
	}
}
