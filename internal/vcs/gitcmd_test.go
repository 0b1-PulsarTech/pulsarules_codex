package vcs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func TestRunGit_FailureNamesTheFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "first", map[string]string{"a.txt": "one"})

	_, err := runGit(dir, "show", "HEAD:nonexistent.go")
	if err == nil {
		t.Fatal("runGit: expected an error for a path missing from HEAD")
	}
	if !strings.Contains(err.Error(), "nonexistent.go") {
		t.Fatalf("err = %v, want it to name the missing file", err)
	}
	if isEmptyRepoError(err) {
		t.Fatalf("isEmptyRepoError(%v) = true, want false (repo has a commit)", err)
	}
}

func TestIsEmptyRepoError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	_, emptyRepoErr := runGit(dir, "log", "-1")
	if emptyRepoErr == nil {
		t.Fatal("runGit: expected an error for log on a repo with no commits")
	}

	writeAndCommit(t, dir, "first", map[string]string{"a.txt": "one"})
	_, otherErr := runGit(dir, "show", "HEAD:nonexistent.go")
	if otherErr == nil {
		t.Fatal("runGit: expected an error for a path missing from HEAD")
	}

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{"empty repo stderr", emptyRepoErr, true},
		{"a different git failure", otherErr, false},
		{"a non-execx error", errors.New("boom"), false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isEmptyRepoError(testCase.err); got != testCase.want {
				t.Errorf("isEmptyRepoError(%v) = %v, want %v", testCase.err, got, testCase.want)
			}
		})
	}
}

func TestRunGitWithin_Timeout(t *testing.T) {
	t.Parallel()

	_, err := runGitWithin(time.Nanosecond, t.TempDir(), "--version")
	if err == nil {
		t.Fatal("runGitWithin: expected an error for an expired timeout")
	}
	if _, ok := errors.AsType[*process.Error](err); !ok {
		t.Fatalf("err = %T, want *execx.Error", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}
