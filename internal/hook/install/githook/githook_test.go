package githook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestInstall_OverwriteReadOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	preCommit := filepath.Join(dir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(preCommit, []byte("old hook"), 0o500); err != nil {
		t.Fatalf("write old hook: %v", err)
	}

	if err := Install(dir, []string{"pre-commit"}); err != nil {
		t.Fatalf("Install over read-only hook: %v", err)
	}

	data, err := os.ReadFile(preCommit)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "governance") {
		t.Errorf("hook not updated: %s", data)
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
