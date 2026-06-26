package output

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteFile asserts parent directories are created and content is written,
// and that a path whose parent is a file surfaces an error.
func TestWriteFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	nested := filepath.Join(dir, "a", "b", "skill.md")
	if err := writeFile(nested, "hello"); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	got, err := os.ReadFile(nested) //nolint:gosec // path is under the test's temp dir.
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "hello" {
		t.Errorf("content = %q, want %q", got, "hello")
	}

	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	if err := writeFile(filepath.Join(blocker, "child.md"), "y"); err == nil {
		t.Error("expected error writing under a file, got nil")
	}
}
