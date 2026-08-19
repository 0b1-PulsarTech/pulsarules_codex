package githook

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

// runGitOrFatal runs one git subcommand in dir, failing the test on error.
func runGitOrFatal(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// newWorktree builds a real repository with one commit plus a linked worktree,
// returning both roots. The worktree's .git is a POINTER FILE, which is the
// condition every assertion below turns on.
func newWorktree(t *testing.T) (mainDir, worktree string) {
	t.Helper()
	root := t.TempDir()
	mainDir = filepath.Join(root, "main")
	if err := os.MkdirAll(mainDir, 0o750); err != nil {
		t.Fatalf("mkdir main: %v", err)
	}
	runGitOrFatal(t, mainDir, "init", "-q")
	if err := os.WriteFile(filepath.Join(mainDir, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	runGitOrFatal(t, mainDir, "add", "-A")
	runGitOrFatal(t, mainDir, "commit", "-q", "-m", "init")

	worktree = filepath.Join(root, "wt")
	runGitOrFatal(t, mainDir, "worktree", "add", "-q", worktree, "-b", "probe")
	entry, err := os.Lstat(filepath.Join(worktree, ".git"))
	if err != nil {
		t.Fatalf("lstat worktree .git: %v", err)
	}
	if entry.IsDir() {
		t.Fatal("worktree .git is a directory, expected a gitdir pointer file")
	}
	return mainDir, worktree
}

// TestInstall_LinkedWorktree asserts an install run from a linked worktree
// lands its hooks in the repository's SHARED hooks dir. Joining the worktree
// root with ".git" cannot work there - the path's parent is a file - so this
// is the case a plain .git/hooks join fails outright.
func TestInstall_LinkedWorktree(t *testing.T) {
	t.Parallel()

	mainDir, worktree := newWorktree(t)
	hooks := []string{"commit-msg", "pre-commit", "pre-push"}
	if _, err := Install(worktree, hooks, Options{}); err != nil {
		t.Fatalf("Install from worktree: %v", err)
	}

	for _, name := range hooks {
		shared := filepath.Join(mainDir, ".git", "hooks", name)
		content, err := os.ReadFile(shared) //nolint:gosec // test fixture path.
		if err != nil {
			t.Fatalf("read shared hook %s: %v", name, err)
		}
		// The script must resolve the binary through the common dir too, or a
		// hook fired from the worktree exits 0 without running anything.
		if !strings.Contains(string(content), "--git-common-dir") {
			t.Errorf("%s: script does not resolve the common dir", name)
		}
		if _, err = os.Lstat(filepath.Join(worktree, ".git", "hooks", name)); err == nil {
			t.Errorf("%s: hook written under the worktree instead of the shared dir", name)
		}
	}
}

// TestUninstall_LinkedWorktree asserts uninstall run from a linked worktree
// removes the hooks and the binary from the shared dir it installed them in.
func TestUninstall_LinkedWorktree(t *testing.T) {
	t.Parallel()

	mainDir, worktree := newWorktree(t)
	if _, err := Install(worktree, []string{"commit-msg"}, Options{}); err != nil {
		t.Fatalf("Install from worktree: %v", err)
	}

	removed, _, err := Uninstall(worktree)
	if err != nil {
		t.Fatalf("Uninstall from worktree: %v", err)
	}
	if !slices.Contains(removed, "commit-msg") {
		t.Errorf("removed = %v, want it to contain commit-msg", removed)
	}
	if _, err = os.Lstat(
		filepath.Join(mainDir, ".git", "hooks", "commit-msg"),
	); !errors.Is(
		err,
		fs.ErrNotExist,
	) {
		t.Errorf("shared hook still present, lstat err = %v", err)
	}
}

// TestOrphans_SurviveUninstall walks the only sequence that strands a backup: a
// foreign hook is displaced twice, so the second rename takes the .1 slot, and
// uninstall restores just the base one. Before this reported, the .1 file sat in
// .git/hooks with nothing telling the operator it existed.
func TestOrphans_SurviveUninstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hooksPath := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksPath, 0o750); err != nil {
		t.Fatalf("mkdir hooks: %v", err)
	}
	hookPath := filepath.Join(hooksPath, "commit-msg")
	writeForeign := func(body string) {
		t.Helper()
		// Install leaves the hook read-only (0o500), so replacing it means
		// unlinking first - exactly what a person editing it by hand would hit.
		_ = os.Remove(hookPath)
		if err := os.WriteFile(hookPath, []byte(body), 0o700); err != nil {
			t.Fatalf("seed foreign hook: %v", err)
		}
	}

	writeForeign("#!/bin/sh\n# first hand-written hook\n")
	if _, err := Install(dir, []string{"commit-msg"}, Options{}); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	writeForeign("#!/bin/sh\n# second hand-written hook\n")
	if _, err := Install(dir, []string{"commit-msg"}, Options{}); err != nil {
		t.Fatalf("second Install: %v", err)
	}

	if _, _, err := Uninstall(dir); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	notes, err := Orphans(dir)
	if err != nil {
		t.Fatalf("Orphans: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("notes = %v, want exactly one", notes)
	}
	if !strings.Contains(notes[0], "commit-msg") {
		t.Errorf("note = %q, want it to name the hook", notes[0])
	}
	// The stranded slot must still hold the content, not just be mentioned.
	stranded, err := os.ReadFile(
		hookPath + marker.BackupSuffix + ".1",
	) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("read stranded slot: %v", err)
	}
	if !strings.Contains(string(stranded), "hand-written hook") {
		t.Errorf("stranded slot content = %q, want the displaced hook", stranded)
	}
}
