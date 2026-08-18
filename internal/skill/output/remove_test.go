package output

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// TestRemoveDocs_RemovesOwnedLeavesForeign asserts RemoveDocs deletes only
// the directories carrying WriteDoc's own fingerprint (docName + the
// matching sibling .gitignore), leaving a user's own directory (no
// .gitignore, or one with different content) untouched.
func TestRemoveDocs_RemovesOwnedLeavesForeign(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := WriteDoc(filepath.Join(dest, "go-style"), "SKILL.md", "body"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if _, err := WriteDoc(
		filepath.Join(dest, "gopls-navigation"),
		"SKILL.md",
		"body2",
	); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	// A user's own directory sharing the dest, with no WriteDoc fingerprint.
	userDir := filepath.Join(dest, "my-own-notes")
	if err := os.MkdirAll(userDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte("mine"), 0o600); err != nil {
		t.Fatalf("seed user file: %v", err)
	}

	removed, _, err := RemoveDocs(dest, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 2 {
		t.Fatalf("removed = %v, want 2 owned ids", removed)
	}
	for _, id := range []string{"go-style", "gopls-navigation"} {
		if _, statErr := os.Stat(filepath.Join(dest, id)); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("expected %q to be removed, stat err = %v", id, statErr)
		}
	}
	if _, statErr := os.Stat(filepath.Join(userDir, "SKILL.md")); statErr != nil {
		t.Errorf("expected user's own file to survive, stat err = %v", statErr)
	}
	// dest itself survives since a foreign directory remains in it.
	if _, statErr := os.Stat(dest); statErr != nil {
		t.Errorf("expected dest to survive (not empty), stat err = %v", statErr)
	}
}

// TestRemoveDocs_RemovesDestWhenEmptied asserts dest itself is removed once
// every entry in it was one RemoveDocs removed.
func TestRemoveDocs_RemovesDestWhenEmptied(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := WriteDoc(filepath.Join(dest, "go-style"), "SKILL.md", "body"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}

	if _, _, err := RemoveDocs(dest, "SKILL.md"); err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}

	if _, statErr := os.Stat(dest); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected dest to be removed once empty, stat err = %v", statErr)
	}
}

// TestRemoveDocs_NoOpWhenAbsent asserts a missing dest is not an error.
func TestRemoveDocs_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	removed, _, err := RemoveDocs(filepath.Join(t.TempDir(), "never-created"), "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("removed = %v, want none", removed)
	}
}

// TestRemoveDocs_PreservesUserFileInsideOwnedDir asserts a file a user (or
// the skill format itself, e.g. references/) placed inside a directory
// RemoveDocs recognizes as WriteDoc's own survives: only the fingerprinted
// docName + .gitignore pair is deleted, leaving the directory non-empty.
// Regression test for the bug where RemoveDocs os.RemoveAll'd the whole dir.
func TestRemoveDocs_PreservesUserFileInsideOwnedDir(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := WriteDoc(filepath.Join(dest, "dataviz"), "SKILL.md", "body"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	nested := filepath.Join(dest, "dataviz", "references", "palette.md")
	if err := os.MkdirAll(filepath.Dir(nested), 0o750); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	if err := os.WriteFile(nested, []byte("palette data"), 0o600); err != nil {
		t.Fatalf("seed nested file: %v", err)
	}

	removed, _, err := RemoveDocs(dest, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 1 || removed[0] != "dataviz" {
		t.Fatalf("removed = %v, want [dataviz]", removed)
	}
	if _, statErr := os.Stat(nested); statErr != nil {
		t.Errorf("expected nested user file to survive, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(
		filepath.Join(dest, "dataviz", "SKILL.md"),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("expected SKILL.md to be removed, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(
		filepath.Join(dest, "dataviz", ".gitignore"),
	); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected .gitignore to be removed, stat err = %v", statErr)
	}
	// The skill directory survives (not empty) because the nested file remains.
	if _, statErr := os.Stat(filepath.Join(dest, "dataviz")); statErr != nil {
		t.Errorf("expected dataviz dir to survive (not empty), stat err = %v", statErr)
	}
}

// TestRemoveDocs_Idempotent asserts running RemoveDocs twice is not an error.
func TestRemoveDocs_Idempotent(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := WriteDoc(filepath.Join(dest, "go-style"), "SKILL.md", "body"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if _, _, err := RemoveDocs(dest, "SKILL.md"); err != nil {
		t.Fatalf("RemoveDocs #1: %v", err)
	}
	if _, _, err := RemoveDocs(dest, "SKILL.md"); err != nil {
		t.Fatalf("RemoveDocs #2: %v", err)
	}
}

// TestRemoveDocs_RestoresBackup asserts removing a doc whose WriteDoc backed
// up a foreign SKILL.md restores that backup to its original path,
// completing WriteDoc's reversal rather than leaving the user's original
// content stranded under its backup name.
func TestRemoveDocs_RestoresBackup(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "security")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	docPath := filepath.Join(dir, "SKILL.md")
	foreign := "# My own security notes\n"
	if err := os.WriteFile(docPath, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign doc: %v", err)
	}
	if _, err := WriteDoc(dir, "SKILL.md", "# rendered\n"+marker.Installed+"\n"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}

	removed, restored, err := RemoveDocs(dest, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 1 || removed[0] != "security" {
		t.Fatalf("removed = %v, want [security]", removed)
	}
	wantMsg := marker.RestoreMessage(docPath)
	if len(restored) != 1 || restored[0] != wantMsg {
		t.Fatalf("restored = %v, want [%q]", restored, wantMsg)
	}
	got, readErr := os.ReadFile(docPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read restored doc: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("restored content = %q, want %q", got, foreign)
	}
}

// TestRemoveDocs_RemovesWithoutGitignore asserts a doc removable purely on
// its content marker.Installed - its sibling .gitignore already deleted, the
// exact scenario WriteDoc's own doc comment invites ("delete that .gitignore
// to commit the doc") - still removes cleanly instead of becoming stuck.
func TestRemoveDocs_RemovesWithoutGitignore(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "go-style")
	if _, err := WriteDoc(dir, "SKILL.md", "# go-style\n"+marker.Installed+"\n"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatalf("remove .gitignore: %v", err)
	}

	removed, _, err := RemoveDocs(dest, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 1 || removed[0] != "go-style" {
		t.Fatalf("removed = %v, want [go-style]", removed)
	}
	if _, statErr := os.Stat(dir); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected dir to be removed, stat err = %v", statErr)
	}
}
