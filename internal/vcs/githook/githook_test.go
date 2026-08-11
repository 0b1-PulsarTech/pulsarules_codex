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
	if err := Install(dir, hooks); err != nil {
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

	if err := Install(t.TempDir(), nil); err != nil {
		t.Fatalf("Install with nil should succeed: %v", err)
	}
}

func TestInstall_UnknownHook(t *testing.T) {
	t.Parallel()

	err := Install(t.TempDir(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown hook")
	}
}

func TestInstall_NoGitDir(t *testing.T) {
	t.Parallel()

	err := Install(t.TempDir(), []string{"commit-msg"})
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

	if err := Install(dir, []string{"pre-commit"}); err != nil {
		t.Fatalf("Install over read-only hook: %v", err)
	}
	if _, statErr := os.Stat(
		preCommit + ownedfile.BackupSuffix,
	); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected no backup (the old hook was ours), stat err = %v", statErr)
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

	if err := Install(dir, []string{"pre-commit"}); err != nil {
		t.Fatalf("Install over foreign hook: %v", err)
	}
	backupPath := preCommit + ownedfile.BackupSuffix

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

	if err := Install(dir, []string{"pre-commit"}); err != nil {
		t.Fatalf("Install over foreign hook: %v", err)
	}
	newBackup := preCommit + ownedfile.BackupSuffix + ".1"
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
