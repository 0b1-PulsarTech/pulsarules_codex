package githook

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func TestInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	hooks := []string{"commit-msg", "pre-commit"}
	if _, err := Install(dir, hooks, Options{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, name := range hooks {
		path := filepath.Join(dir, ".git", "hooks", name)
		hookContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(hookContent), "pulsarules_cli") {
			t.Errorf("%s: missing binary reference", name)
		}
		stat, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if stat.Mode()&0o111 == 0 {
			t.Errorf("%s: not executable", name)
		}
	}
}

func TestInstall_NoHooks(t *testing.T) {
	t.Parallel()

	if _, err := Install(t.TempDir(), nil, Options{}); err != nil {
		t.Fatalf("Install with nil should succeed: %v", err)
	}
}

func TestInstall_UnknownHook(t *testing.T) {
	t.Parallel()

	_, err := Install(t.TempDir(), []string{"nonexistent"}, Options{})
	if err == nil {
		t.Fatal("expected error for unknown hook")
	}
}

func TestInstall_NoGitDir(t *testing.T) {
	t.Parallel()

	_, err := Install(t.TempDir(), []string{"commit-msg"}, Options{})
	if err != nil {
		t.Fatalf("expected dir to be created: %v", err)
	}
}

// TestInstall_OverwriteReadOnly asserts a read-only hook Install itself wrote
// earlier (it carries marker.Installed) is overwritten in place - proving
// os.Remove clears the 0o500 mode WriteFile's O_TRUNC cannot - and reports no
// backup, since it is ours to begin with.
func TestInstall_OverwriteReadOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	oldOurs := "#!/bin/sh\n# " + marker.Installed + "\necho old\n"
	if err := os.WriteFile(preCommit, []byte(oldOurs), 0o500); err != nil {
		t.Fatalf("write old hook: %v", err)
	}

	backedUp, err := Install(dir, []string{"pre-commit"}, Options{})
	if err != nil {
		t.Fatalf("Install over read-only hook: %v", err)
	}
	if len(backedUp) != 0 {
		t.Errorf("backedUp = %v, want none (the old hook was ours)", backedUp)
	}

	updatedContent, err := os.ReadFile(preCommit)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(updatedContent), "governance") {
		t.Errorf("hook not updated: %s", updatedContent)
	}
}

// TestInstall_BacksUpForeignHook is the regression test for the data-loss
// defect: a hand-written pre-commit hook (a husky-style "npx lint-staged"
// script, carrying no marker.Installed) is renamed to a
// ".pulsarules-backup" slot - never destroyed - before Install writes its
// own script over the path, and the rename is reported through backedUp.
func TestInstall_BacksUpForeignHook(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	foreign := "#!/bin/sh\nnpx lint-staged\n"
	if err := os.WriteFile(preCommit, []byte(foreign), 0o755); err != nil {
		t.Fatalf("seed foreign hook: %v", err)
	}

	backedUp, err := Install(dir, []string{"pre-commit"}, Options{})
	if err != nil {
		t.Fatalf("Install over foreign hook: %v", err)
	}
	backupPath := preCommit + marker.BackupSuffix
	wantMsg := marker.BackupMessage(preCommit, backupPath)
	if len(backedUp) != 1 || backedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", backedUp, wantMsg)
	}

	gotBackup, err := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(gotBackup) != foreign {
		t.Errorf("backup content = %q, want %q", gotBackup, foreign)
	}

	installedContent, err := os.ReadFile(preCommit)
	if err != nil {
		t.Fatalf("read installed hook: %v", err)
	}
	if !strings.Contains(string(installedContent), "governance") {
		t.Errorf("hook not installed: %s", installedContent)
	}
}

// TestInstall_NeverOverwritesExistingBackup asserts installing over a
// foreign hook when a ".pulsarules-backup" slot is already occupied (by an
// earlier, unresolved backup) falls back to the next free numbered slot
// instead of clobbering it.
func TestInstall_NeverOverwritesExistingBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	current := "#!/bin/sh\nnpx lint-staged --current\n"
	if err := os.WriteFile(preCommit, []byte(current), 0o755); err != nil {
		t.Fatalf("seed foreign hook: %v", err)
	}
	priorBackup := preCommit + marker.BackupSuffix
	priorContent := "#!/bin/sh\nnpx lint-staged --even-older\n"
	if err := os.WriteFile(priorBackup, []byte(priorContent), 0o600); err != nil {
		t.Fatalf("seed prior backup: %v", err)
	}

	backedUp, err := Install(dir, []string{"pre-commit"}, Options{})
	if err != nil {
		t.Fatalf("Install over foreign hook: %v", err)
	}
	newBackup := preCommit + marker.BackupSuffix + ".1"
	wantMsg := marker.BackupMessage(preCommit, newBackup)
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

// TestUninstall asserts Install then Uninstall removes the fingerprinted
// hooks and the installer binary, but leaves an unrelated pre-push hook a
// user wrote themselves (no marker.Installed) untouched.
func TestUninstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if _, err := Install(dir, []string{"commit-msg", "pre-commit"}, Options{}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if err := InstallBinary(dir); err != nil {
		t.Fatalf("InstallBinary: %v", err)
	}
	foreign := filepath.Join(dir, ".git", "hooks", "pre-push")
	if err := os.WriteFile(foreign, []byte("#!/bin/sh\necho mine\n"), 0o700); err != nil {
		t.Fatalf("seed foreign hook: %v", err)
	}

	removed, _, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	wantRemoved := []string{"commit-msg", "pre-commit"}
	for _, name := range wantRemoved {
		if _, statErr := os.Stat(
			filepath.Join(dir, ".git", "hooks", name),
		); !errors.Is(statErr, fs.ErrNotExist) {
			t.Errorf("expected %q to be removed, stat err = %v", name, statErr)
		}
	}
	if len(removed) != len(wantRemoved) {
		t.Errorf("removed = %v, want 2 hooks", removed)
	}
	if _, statErr := os.Stat(
		filepath.Join(dir, ".git", "hooks", "pulsarules_cli"),
	); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected installer binary to be removed, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(foreign); statErr != nil {
		t.Errorf("expected foreign pre-push hook to survive, stat err = %v", statErr)
	}
}

// TestUninstall_Idempotent asserts running Uninstall against a directory
// Install never touched is not an error.
func TestUninstall_Idempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, _, err := Uninstall(dir); err != nil {
		t.Fatalf("Uninstall on untouched dir: %v", err)
	}
}

// TestUninstall_RestoresBackup asserts uninstalling a hook whose Install
// backed up a foreign file restores that backup to the hook's original
// path, completing the reversal rather than leaving the user's original
// content stranded under its backup name.
func TestUninstall_RestoresBackup(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	foreign := "#!/bin/sh\nnpx lint-staged\n"
	if err := os.WriteFile(preCommit, []byte(foreign), 0o755); err != nil {
		t.Fatalf("seed foreign hook: %v", err)
	}
	if _, err := Install(dir, []string{"pre-commit"}, Options{}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	removed, restored, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(removed) != 1 || removed[0] != "pre-commit" {
		t.Fatalf("removed = %v, want [pre-commit]", removed)
	}
	wantMsg := marker.RestoreMessage(preCommit)
	if len(restored) != 1 || restored[0] != wantMsg {
		t.Fatalf("restored = %v, want [%q]", restored, wantMsg)
	}

	got, readErr := os.ReadFile(preCommit) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read restored hook: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("restored content = %q, want %q", got, foreign)
	}
	if _, statErr := os.Stat(preCommit + marker.BackupSuffix); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected backup slot to be gone after restore, stat err = %v", statErr)
	}
}

func TestHookNames(t *testing.T) {
	t.Parallel()

	names := HookNames()
	if len(names) == 0 {
		t.Fatal("expected at least one hook name")
	}
	hasCommitMsg := false
	for _, name := range names {
		if name == "commit-msg" {
			hasCommitMsg = true
		}
	}
	if !hasCommitMsg {
		t.Error("expected commit-msg in hook names")
	}
}
