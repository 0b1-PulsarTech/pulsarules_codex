package docs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

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

func TestWriteDoc_BacksUpForeignDir(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "security")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	docPath := filepath.Join(dir, "SKILL.md")
	foreign := "# My own security notes\nDo not touch.\n"
	if err := os.WriteFile(docPath, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign doc: %v", err)
	}

	backedUp, err := Write(dir, "SKILL.md", "# rendered body\n"+ownedfile.Marker+"\n")
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	backupPath := docPath + ownedfile.BackupSuffix
	wantMsg := ownedfile.BackupMessage(docPath, backupPath)
	if len(backedUp) != 1 || backedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", backedUp, wantMsg)
	}
	got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("backup content = %q, want %q", got, foreign)
	}
	installed, readErr := os.ReadFile(docPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read installed doc: %v", readErr)
	}
	if !strings.Contains(string(installed), ownedfile.Marker) {
		t.Errorf("installed doc missing marker: %q", installed)
	}
}

func TestWriteDoc_OwnDirOverwrittenWithoutBackup(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "go-style")
	body := "# go-style\n" + ownedfile.Marker + "\nfirst render\n"
	if _, err := Write(dir, "SKILL.md", body); err != nil {
		t.Fatalf("first WriteDoc: %v", err)
	}

	updated := "# go-style\n" + ownedfile.Marker + "\nsecond render\n"
	backedUp, err := Write(dir, "SKILL.md", updated)
	if err != nil {
		t.Fatalf("second WriteDoc: %v", err)
	}
	if len(backedUp) != 0 {
		t.Errorf("backedUp = %v, want none (the dir was already ours)", backedUp)
	}
	got, readErr := os.ReadFile(filepath.Join(dir, "SKILL.md")) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read doc: %v", readErr)
	}
	if string(got) != updated {
		t.Errorf("doc content = %q, want %q", got, updated)
	}
	if _, statErr := os.Stat(
		filepath.Join(dir, "SKILL.md"+ownedfile.BackupSuffix),
	); statErr == nil {
		t.Error("expected no backup file to exist")
	}
}

func TestWriteDoc_NextNumberedSlot(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "security")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	docPath := filepath.Join(dir, "SKILL.md")
	current := "# current foreign notes\n"
	if err := os.WriteFile(docPath, []byte(current), 0o600); err != nil {
		t.Fatalf("seed foreign doc: %v", err)
	}
	priorBackup := docPath + ownedfile.BackupSuffix
	priorContent := "# an even earlier version of the user's notes\n"
	if err := os.WriteFile(priorBackup, []byte(priorContent), 0o600); err != nil {
		t.Fatalf("seed prior backup: %v", err)
	}

	backedUp, err := Write(dir, "SKILL.md", "# rendered body\n"+ownedfile.Marker+"\n")
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	newBackup := docPath + ownedfile.BackupSuffix + ".1"
	wantMsg := ownedfile.BackupMessage(docPath, newBackup)
	if len(backedUp) != 1 || backedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", backedUp, wantMsg)
	}
	got, readErr := os.ReadFile(priorBackup) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read prior backup: %v", readErr)
	}
	if string(got) != priorContent {
		t.Errorf("prior backup content = %q, want %q (clobbered)", got, priorContent)
	}
	got, readErr = os.ReadFile(newBackup) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read new backup: %v", readErr)
	}
	if string(got) != current {
		t.Errorf("new backup content = %q, want %q", got, current)
	}
}

func TestWriteDoc_PreMarkerDoc(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "go-style")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	docPath := filepath.Join(dir, "SKILL.md")
	legacyDoc := "# go-style\nno marker here\n"
	if err := os.WriteFile(docPath, []byte(legacyDoc), 0o600); err != nil {
		t.Fatalf("seed legacy doc: %v", err)
	}
	gitignorePath := filepath.Join(dir, ".gitignore")
	legacyGitignore := "SKILL.md\n.gitignore\n"
	if err := os.WriteFile(gitignorePath, []byte(legacyGitignore), 0o600); err != nil {
		t.Fatalf("seed legacy gitignore: %v", err)
	}

	backedUp, err := Write(dir, "SKILL.md", "# go-style\n"+ownedfile.Marker+"\n")
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	wantDocMsg := ownedfile.BackupMessage(docPath, docPath+ownedfile.BackupSuffix)
	wantGitignoreMsg := ownedfile.BackupMessage(gitignorePath, gitignorePath+ownedfile.BackupSuffix)
	if len(backedUp) != 2 || backedUp[0] != wantDocMsg || backedUp[1] != wantGitignoreMsg {
		t.Fatalf(
			"backedUp = %v, want [%q %q] (the legacy fingerprint no longer proves ownership)",
			backedUp, wantDocMsg, wantGitignoreMsg,
		)
	}
	got, readErr := os.ReadFile(docPath + ownedfile.BackupSuffix) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read doc backup: %v", readErr)
	}
	if string(got) != legacyDoc {
		t.Errorf("doc backup content = %q, want %q", got, legacyDoc)
	}
}
