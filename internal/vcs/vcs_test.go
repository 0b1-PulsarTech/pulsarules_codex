package vcs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func commitEnv(authorDate string) []string {
	if authorDate == "" {
		authorDate = "2024-01-01T00:00:00Z"
	}
	return append(
		os.Environ(),
		"GIT_AUTHOR_DATE="+authorDate,
		"GIT_COMMITTER_DATE="+authorDate,
	)
}

func runGitOrFatal(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(
		context.Background(), "git", append([]string{"-C", dir}, args...)...,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initRepo(t *testing.T, dir string) {
	t.Helper()
	runGitOrFatal(t, dir, "init", "-q")
	runGitOrFatal(t, dir, "config", "user.email", "test@example.com")
	runGitOrFatal(t, dir, "config", "user.name", "Test")
}

func writeAndCommit(t *testing.T, dir, msg string, files map[string]string) {
	t.Helper()
	for path, content := range files {
		writeFile(t, dir, path, content)
	}
	runGitOrFatal(t, dir, "add", "-A")

	cmd := exec.CommandContext(context.Background(), "git", "-C", dir, "commit", "-q", "-m", msg)
	cmd.Env = commitEnv("")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func TestOpen_NotARepository(t *testing.T) {
	t.Parallel()

	_, err := Open(t.TempDir())
	if !errors.Is(err, ErrNoRepository) {
		t.Fatalf("Open() err = %v, want ErrNoRepository", err)
	}
}

func TestRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	wantRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", dir, err)
	}
	gotRoot, err := filepath.EvalSymlinks(repo.Root())
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", repo.Root(), err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("Root() = %q, want %q", gotRoot, wantRoot)
	}
}

func TestOpen_LinkedWorktree(t *testing.T) {
	t.Parallel()

	mainDir := t.TempDir()
	initRepo(t, mainDir)
	writeAndCommit(t, mainDir, "main commit", map[string]string{"a.txt": "one"})

	parent := t.TempDir()
	wtDir := filepath.Join(parent, "linked-wt")
	runGitOrFatal(t, mainDir, "worktree", "add", "-q", wtDir, "-b", "feature-branch")

	repo, err := Open(wtDir)
	if err != nil {
		t.Fatalf("Open(linked worktree): %v", err)
	}

	wantRoot, err := filepath.EvalSymlinks(wtDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", wtDir, err)
	}
	gotRoot, err := filepath.EvalSymlinks(repo.Root())
	if err != nil {
		t.Fatalf("EvalSymlinks(%s): %v", repo.Root(), err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("Root() = %q, want %q", gotRoot, wantRoot)
	}
}
