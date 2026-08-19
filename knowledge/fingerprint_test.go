package knowledge

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestFingerprint_Stable asserts the embedded digest is deterministic: an unstable
// one would report drift on every session and train people to ignore the warning.
func TestFingerprint_Stable(t *testing.T) {
	t.Parallel()

	first, err := Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint: %v", err)
	}
	second, err := Fingerprint()
	if err != nil {
		t.Fatalf("Fingerprint again: %v", err)
	}
	if first != second {
		t.Errorf("Fingerprint is unstable: %q then %q", first, second)
	}
	if first == "" {
		t.Error("Fingerprint is empty")
	}
}

// TestFingerprintDir asserts an on-disk tree is comparable to the embedded one,
// that a byte-level edit changes the digest, and that a missing tree surfaces as
// fs.ErrNotExist so a caller can tell "nothing to compare" from "cannot read".
func TestFingerprintDir(t *testing.T) {
	t.Parallel()

	t.Run("matches the embedded tree it was copied from", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		copyEmbeddedStandards(t, root)
		onDisk, err := FingerprintDir(root)
		if err != nil {
			t.Fatalf("FingerprintDir: %v", err)
		}
		embedded, err := Fingerprint()
		if err != nil {
			t.Fatalf("Fingerprint: %v", err)
		}
		if onDisk != embedded {
			t.Errorf("on-disk digest %q does not match embedded %q", onDisk, embedded)
		}
	})

	t.Run("an edited byte changes the digest", func(t *testing.T) {
		t.Parallel()

		root := t.TempDir()
		copyEmbeddedStandards(t, root)
		before, err := FingerprintDir(root)
		if err != nil {
			t.Fatalf("FingerprintDir: %v", err)
		}
		edited := filepath.Join(root, standardsRoot, "skills.yaml")
		body, err := os.ReadFile(edited) //nolint:gosec // path built from t.TempDir.
		if err != nil {
			t.Fatalf("read skills.yaml: %v", err)
		}
		if err = os.WriteFile(edited, append(body, '\n'), 0o600); err != nil {
			t.Fatalf("edit skills.yaml: %v", err)
		}
		after, err := FingerprintDir(root)
		if err != nil {
			t.Fatalf("FingerprintDir after edit: %v", err)
		}
		if before == after {
			t.Error("digest did not change after editing a standards file")
		}
	})

	t.Run("a missing tree reports fs.ErrNotExist", func(t *testing.T) {
		t.Parallel()

		if _, err := FingerprintDir(t.TempDir()); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("err = %v, want fs.ErrNotExist", err)
		}
	})
}

// copyEmbeddedStandards writes the embedded standards tree under root, so an
// on-disk digest can be compared against the embedded one byte for byte.
func copyEmbeddedStandards(t *testing.T, root string) {
	t.Helper()
	err := fs.WalkDir(
		standardsFS,
		standardsRoot,
		func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			target := filepath.Join(root, path)
			if entry.IsDir() {
				return os.MkdirAll(target, 0o750)
			}
			content, readErr := fs.ReadFile(standardsFS, path)
			if readErr != nil {
				return readErr
			}
			return os.WriteFile(target, content, 0o600)
		},
	)
	if err != nil {
		t.Fatalf("copy embedded standards: %v", err)
	}
}
