package githook

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func TestInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	hooks := []string{"commit-msg", "pre-commit"}
	if _, err := Install(dir, hooks); err != nil {
		t.Fatalf("Install: %v", err)
	}

	for _, name := range hooks {
		path := filepath.Join(dir, ".git", "hooks", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(data), "pulsarules_cli") {
			t.Errorf("%s: missing binary reference", name)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("%s: not executable", name)
		}
	}
}

func TestInstall_NoHooks(t *testing.T) {
	t.Parallel()

	if _, err := Install(t.TempDir(), nil); err != nil {
		t.Fatalf("Install with nil should succeed: %v", err)
	}
}

func TestInstall_UnknownHook(t *testing.T) {
	t.Parallel()

	_, err := Install(t.TempDir(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown hook")
	}
}

func TestInstall_NoGitDir(t *testing.T) {
	t.Parallel()

	_, err := Install(t.TempDir(), []string{"commit-msg"})
	if err != nil {
		t.Fatalf("expected dir to be created: %v", err)
	}
}

func TestInstall_OwnReadOnlyHook(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	oldOurs := "#!/bin/sh\n# " + ownedfile.Marker + "\necho old\n"
	if err := os.WriteFile(preCommit, []byte(oldOurs), 0o500); err != nil {
		t.Fatalf("write old hook: %v", err)
	}

	backedUp, err := Install(dir, []string{"pre-commit"})
	if err != nil {
		t.Fatalf("Install over read-only hook: %v", err)
	}
	if len(backedUp) != 0 {
		t.Errorf("backedUp = %v, want none (the old hook was ours)", backedUp)
	}

	data, err := os.ReadFile(preCommit)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "governance") {
		t.Errorf("hook not updated: %s", data)
	}
}

func TestInstall_ForeignHookBackup(t *testing.T) {
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

	backedUp, err := Install(dir, []string{"pre-commit"})
	if err != nil {
		t.Fatalf("Install over foreign hook: %v", err)
	}
	backupPath := preCommit + ownedfile.BackupSuffix
	wantMsg := ownedfile.BackupMessage(preCommit, backupPath)
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

	data, err := os.ReadFile(preCommit)
	if err != nil {
		t.Fatalf("read installed hook: %v", err)
	}
	if !strings.Contains(string(data), "governance") {
		t.Errorf("hook not installed: %s", data)
	}
}

func TestInstall_ExistingBackupNextSlot(t *testing.T) {
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
	priorBackup := preCommit + ownedfile.BackupSuffix
	priorContent := "#!/bin/sh\nnpx lint-staged --even-older\n"
	if err := os.WriteFile(priorBackup, []byte(priorContent), 0o600); err != nil {
		t.Fatalf("seed prior backup: %v", err)
	}

	backedUp, err := Install(dir, []string{"pre-commit"})
	if err != nil {
		t.Fatalf("Install over foreign hook: %v", err)
	}
	newBackup := preCommit + ownedfile.BackupSuffix + ".1"
	wantMsg := ownedfile.BackupMessage(preCommit, newBackup)
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

func TestUninstall_FingerprintedHooks(
	t *testing.T,
) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if _, err := Install(dir, []string{"commit-msg", "pre-commit"}); err != nil {
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

func TestUninstall_DirectoryInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if _, _, err := Uninstall(dir); err != nil {
		t.Fatalf("Uninstall on untouched dir: %v", err)
	}
}

func TestUninstall_RestoresBackup(
	t *testing.T,
) {
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
	if _, err := Install(dir, []string{"pre-commit"}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	removed, restored, err := Uninstall(dir)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(removed) != 1 || removed[0] != "pre-commit" {
		t.Fatalf("removed = %v, want [pre-commit]", removed)
	}
	wantMsg := ownedfile.RestoreMessage(preCommit)
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
	if _, statErr := os.Stat(
		preCommit + ownedfile.BackupSuffix,
	); !errors.Is(statErr, fs.ErrNotExist) {
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
