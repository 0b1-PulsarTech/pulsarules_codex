package docs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func TestRemoveDocs_RemovesOwnedLeavesForeign(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := Write(
		filepath.Join(dest, "go-style"), "SKILL.md", "body\n"+ownedfile.Marker+"\n",
	); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if _, err := Write(
		filepath.Join(dest, "gopls-navigation"),
		"SKILL.md",
		"body2\n"+ownedfile.Marker+"\n",
	); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	userDir := filepath.Join(dest, "my-own-notes")
	if err := os.MkdirAll(userDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "SKILL.md"), []byte("mine"), 0o600); err != nil {
		t.Fatalf("seed user file: %v", err)
	}

	removed, _, err := Remove(dest, "SKILL.md")
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
	if _, statErr := os.Stat(dest); statErr != nil {
		t.Errorf("expected dest to survive (not empty), stat err = %v", statErr)
	}
}

func TestRemoveDocs_RemovesDestWhenEmptied(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := Write(
		filepath.Join(dest, "go-style"), "SKILL.md", "body\n"+ownedfile.Marker+"\n",
	); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}

	if _, _, err := Remove(dest, "SKILL.md"); err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}

	if _, statErr := os.Stat(dest); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected dest to be removed once empty, stat err = %v", statErr)
	}
}

func TestRemoveDocs_NoOpWhenAbsent(t *testing.T) {
	t.Parallel()

	removed, _, err := Remove(filepath.Join(t.TempDir(), "never-created"), "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("removed = %v, want none", removed)
	}
}

func TestRemoveDocs_PreservesUserFileInsideOwnedDir(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := Write(
		filepath.Join(dest, "dataviz"), "SKILL.md", "body\n"+ownedfile.Marker+"\n",
	); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	nested := filepath.Join(dest, "dataviz", "references", "palette.md")
	if err := os.MkdirAll(filepath.Dir(nested), 0o750); err != nil {
		t.Fatalf("mkdir references: %v", err)
	}
	if err := os.WriteFile(nested, []byte("palette data"), 0o600); err != nil {
		t.Fatalf("seed nested file: %v", err)
	}

	removed, _, err := Remove(dest, "SKILL.md")
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
	if _, statErr := os.Stat(filepath.Join(dest, "dataviz")); statErr != nil {
		t.Errorf("expected dataviz dir to survive (not empty), stat err = %v", statErr)
	}
}

func TestRemoveDocs_Idempotent(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	if _, err := Write(
		filepath.Join(dest, "go-style"), "SKILL.md", "body\n"+ownedfile.Marker+"\n",
	); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if _, _, err := Remove(dest, "SKILL.md"); err != nil {
		t.Fatalf("RemoveDocs #1: %v", err)
	}
	if _, _, err := Remove(dest, "SKILL.md"); err != nil {
		t.Fatalf("RemoveDocs #2: %v", err)
	}
}

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
	if _, err := Write(dir, "SKILL.md", "# rendered\n"+ownedfile.Marker+"\n"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}

	removed, restored, err := Remove(dest, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(removed) != 1 || removed[0] != "security" {
		t.Fatalf("removed = %v, want [security]", removed)
	}
	wantMsg := ownedfile.RestoreMessage(docPath)
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

func TestRemoveDocs_RemovesWithoutGitignore(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "go-style")
	if _, err := Write(dir, "SKILL.md", "# go-style\n"+ownedfile.Marker+"\n"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if err := os.Remove(filepath.Join(dir, ".gitignore")); err != nil {
		t.Fatalf("remove .gitignore: %v", err)
	}

	removed, _, err := Remove(dest, "SKILL.md")
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

func TestRemoveDocs_LeavesUserGitignoreAndSkipsRestore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "security")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(ignorePath, []byte("keep-me\n"), 0o600); err != nil {
		t.Fatalf("seed gitignore: %v", err)
	}
	if _, err := Write(dir, "SKILL.md", "body\n"+ownedfile.Marker+"\n"); err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	backupPath := ignorePath + ownedfile.BackupSuffix
	rendered, err := os.ReadFile(ignorePath) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read gitignore: %v", err)
	}
	if err = os.WriteFile(ignorePath, append(rendered, "mine/\n"...), 0o600); err != nil {
		t.Fatalf("add user line: %v", err)
	}

	_, restored, err := Remove(root, "SKILL.md")
	if err != nil {
		t.Fatalf("RemoveDocs: %v", err)
	}
	if len(restored) != 0 {
		t.Errorf("restored = %v, want none while a user gitignore stands", restored)
	}
	for path, want := range map[string]string{ignorePath: "mine/\n", backupPath: "keep-me\n"} {
		got, readErr := os.ReadFile(path) //nolint:gosec // test fixture.
		if readErr != nil {
			t.Fatalf("read %q: %v", path, readErr)
		}
		if string(got) != want {
			t.Errorf("%q content = %q, want %q", path, got, want)
		}
	}
}
