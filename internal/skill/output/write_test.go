package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
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
	if err = os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	if err = writeFile(filepath.Join(blocker, "child.md"), "y"); err == nil {
		t.Error("expected error writing under a file, got nil")
	}
}

// TestWriteDoc_BacksUpForeignDir is the regression test for the data-loss
// defect: a hand-written skill directory (a SKILL.md with no marker.Installed
// and no sibling .gitignore - a name a user could plausibly already own, e.g.
// "security") is backed up rather than destroyed when WriteDoc installs over
// it, and the rename is reported through backedUp.
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

	backedUp, err := WriteDoc(dir, "SKILL.md", "# rendered body\n"+marker.Installed+"\n")
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	backupPath := docPath + marker.BackupSuffix
	wantMsg := marker.BackupMessage(docPath, backupPath)
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
	if !strings.Contains(string(installed), marker.Installed) {
		t.Errorf("installed doc missing marker: %q", installed)
	}
}

// TestWriteDoc_OwnDirOverwrittenWithoutBackup asserts a second WriteDoc call
// against a directory it already owns (proven by the rendered body's own
// marker.Installed) overwrites in place with no backup and no spurious
// warning.
func TestWriteDoc_OwnDirOverwrittenWithoutBackup(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "go-style")
	body := "# go-style\n" + marker.Installed + "\nfirst render\n"
	if _, err := WriteDoc(dir, "SKILL.md", body); err != nil {
		t.Fatalf("first WriteDoc: %v", err)
	}

	updated := "# go-style\n" + marker.Installed + "\nsecond render\n"
	backedUp, err := WriteDoc(dir, "SKILL.md", updated)
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
	if _, statErr := os.Stat(filepath.Join(dir, "SKILL.md"+marker.BackupSuffix)); statErr == nil {
		t.Error("expected no backup file to exist")
	}
}

// TestWriteDoc_NeverOverwritesExistingBackup asserts a foreign doc backed up
// when a ".pulsarules-backup" slot is already occupied (by an earlier,
// unresolved backup) falls back to the next free numbered slot instead of
// clobbering it.
func TestWriteDoc_NeverOverwritesExistingBackup(t *testing.T) {
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
	priorBackup := docPath + marker.BackupSuffix
	priorContent := "# an even earlier version of the user's notes\n"
	if err := os.WriteFile(priorBackup, []byte(priorContent), 0o600); err != nil {
		t.Fatalf("seed prior backup: %v", err)
	}

	backedUp, err := WriteDoc(dir, "SKILL.md", "# rendered body\n"+marker.Installed+"\n")
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	newBackup := docPath + marker.BackupSuffix + ".1"
	wantMsg := marker.BackupMessage(docPath, newBackup)
	if len(backedUp) != 1 || backedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", backedUp, wantMsg)
	}
	// The prior backup must survive untouched.
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

// TestWriteDoc_LegacyGitignoreOwnershipStillRecognized asserts a doc rendered
// before the content marker existed - proven only by the sibling .gitignore
// fingerprint - is still recognized as this tool's own and overwritten in
// place with no backup, so upgrading to the marker-carrying template does
// not treat every previously-installed project as foreign.
func TestWriteDoc_LegacyGitignoreOwnershipStillRecognized(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	dir := filepath.Join(dest, "go-style")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "SKILL.md"), []byte("# go-style\nno marker here\n"), 0o600,
	); err != nil {
		t.Fatalf("seed legacy doc: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, ".gitignore"), []byte("SKILL.md\n.gitignore\n"), 0o600,
	); err != nil {
		t.Fatalf("seed legacy gitignore: %v", err)
	}

	backedUp, err := WriteDoc(dir, "SKILL.md", "# go-style\n"+marker.Installed+"\n")
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}
	if len(backedUp) != 0 {
		t.Errorf(
			"backedUp = %v, want none (recognized via the legacy .gitignore fingerprint)",
			backedUp,
		)
	}
}

// TestWriteDoc_GitignoreFailureLeavesTheDocUntouched pins the write ORDER. The
// target dir is read-only, so CREATING the .gitignore fails while overwriting the
// already-present doc still succeeds - dir write permission gates creation, not
// writing through an existing file. The old order rewrote the doc first.
func TestWriteDoc_GitignoreFailureLeavesTheDocUntouched(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "skill")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	// The marker makes the doc owned, so WriteDoc skips the backup step and
	// goes straight to the writes this test is about.
	const sentinel = "# " + marker.Installed + "\nprevious body\n"
	docPath := filepath.Join(target, "SKILL.md")
	if err := os.WriteFile(docPath, []byte(sentinel), 0o600); err != nil {
		t.Fatalf("seed doc: %v", err)
	}
	if err := os.Chmod(target, 0o500); err != nil {
		t.Fatalf("chmod read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(target, 0o700) })

	if _, err := WriteDoc(target, "SKILL.md", "new body"); err == nil {
		t.Skip("directory permissions are not enforced for this user")
	}
	got, err := os.ReadFile(docPath) //nolint:gosec // test fixture path.
	if err != nil {
		t.Fatalf("read doc: %v", err)
	}
	if string(got) != sentinel {
		t.Errorf("doc was rewritten before the ignore landed: content = %q, want %q", got, sentinel)
	}
}
