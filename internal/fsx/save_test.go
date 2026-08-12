package fsx

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestSave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	value := map[string]string{"key": "value"}

	if err := Save(path, value); err != nil {
		t.Fatalf("Save: %v", err)
	}

	//nolint:gosec // temp dir.
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "{\n  \"key\": \"value\"\n}\n"
	if string(got) != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestSave_NaN(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := Save(path, math.NaN()); err == nil {
		t.Fatal("Save(NaN) err = nil, want a marshal error")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("Save wrote a file despite the marshal error")
	}
}

func TestSave_MissingParentDirSurfacesWriteError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing-parent", "config.json")

	if err := Save(path, map[string]string{"key": "value"}); err == nil {
		t.Fatal("Save into a missing parent dir err = nil, want a write error")
	}
}

func TestSave_FailedWriteLeavesOriginalIntact(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	original := []byte(`{"kept":"as-is"}` + "\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("seed original: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("restrict dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if err := Save(path, map[string]string{"new": "value"}); err == nil {
		t.Fatal("Save into a read-only dir err = nil, want a temp-file error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after failed Save: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("content = %q, want unchanged %q", got, original)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf(
			"dir entries = %v, want only the original config file (no leftover temp file)",
			entries,
		)
	}
}
