package vcs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func setupNormal(t *testing.T, root string) (dir, want string) {
	t.Helper()
	initRepo(t, root)
	return root, filepath.Join(root, ".git")
}

func setupLinked(t *testing.T, root string) (dir, want string) {
	t.Helper()
	mainDir := filepath.Join(root, "main")
	if err := os.MkdirAll(mainDir, 0o750); err != nil {
		t.Fatalf("mkdir main: %v", err)
	}
	initRepo(t, mainDir)
	writeAndCommit(t, mainDir, "init", map[string]string{"a.txt": "a"})
	worktree := filepath.Join(root, "wt")
	runGitOrFatal(t, mainDir, "worktree", "add", "-q", worktree, "-b", "probe")
	entry, statErr := os.Lstat(filepath.Join(worktree, ".git"))
	if statErr != nil {
		t.Fatalf("lstat worktree .git: %v", statErr)
	}
	if entry.IsDir() {
		t.Fatal("worktree .git is a directory, expected a gitdir pointer file")
	}
	return worktree, filepath.Join(mainDir, ".git")
}

func setupBare(t *testing.T, root string) (dir, want string) {
	t.Helper()
	return root, ""
}

func setupBroken(t *testing.T, root string) (dir, want string) {
	t.Helper()
	pointer := []byte("gitdir: " + filepath.Join(root, "gone") + "\n")
	if err := os.WriteFile(filepath.Join(root, ".git"), pointer, 0o600); err != nil {
		t.Fatalf("write pointer: %v", err)
	}
	return root, ""
}

func TestCommonDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, root string) (dir, want string)
		wantErr      error
		unclassified bool
	}{
		{name: "normal checkout resolves its own git dir", setup: setupNormal},
		{name: "linked worktree resolves the main repository git dir", setup: setupLinked},
		{name: "outside a repository", setup: setupBare, wantErr: ErrNoRepository},
		{
			name:         "worktree pointer to nowhere is not reported as no repository",
			setup:        setupBroken,
			unclassified: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir, want := testCase.setup(t, t.TempDir())
			got, err := CommonDir(dir)
			if testCase.unclassified {
				assertUnclassified(t, err)
				return
			}
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("CommonDir(%q) err = %v, want %v", dir, err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CommonDir(%q): %v", dir, err)
			}
			if resolved(t, got) != resolved(t, want) {
				t.Errorf("CommonDir(%q) = %q, want %q", dir, got, want)
			}
		})
	}
}

func resolved(t *testing.T, path string) string {
	t.Helper()
	out, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve %q: %v", path, err)
	}
	return out
}

func assertUnclassified(t *testing.T, err error) {
	t.Helper()
	if err == nil || errors.Is(err, ErrNoRepository) || errors.Is(err, ErrGitNotFound) {
		t.Fatalf("err = %v, want an error that is neither sentinel", err)
	}
}

func TestClassifyCommonDir(t *testing.T) {
	t.Parallel()

	const noRepo = "fatal: not a git repository (or any parent up to mount point /)"
	testCases := []struct {
		name         string
		err          error
		want         error
		unclassified bool
	}{
		{name: "git missing", err: &process.Error{Err: exec.ErrNotFound}, want: ErrGitNotFound},
		{name: "no repository", err: &process.Error{Stderr: noRepo}, want: ErrNoRepository},
		{name: "timeout", err: &process.Error{Err: context.DeadlineExceeded}, unclassified: true},
		{
			name:         "gitdir pointer to nowhere",
			err:          &process.Error{Stderr: "fatal: not a git repository: /x/gone"},
			unclassified: true,
		},
		{
			name:         "dubious ownership",
			err:          &process.Error{Stderr: "fatal: detected dubious ownership"},
			unclassified: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := classifyCommonDir(testCase.err)
			if testCase.unclassified {
				assertUnclassified(t, got)
				return
			}
			if !errors.Is(got, testCase.want) {
				t.Fatalf("classifyCommonDir() = %v, want %v", got, testCase.want)
			}
		})
	}
}
