package vcs

import (
	"errors"
	"strings"
	"testing"
)

// TestRunGit_FailureNamesTheFile proves a git failure unrelated to an empty
// repository propagates as a gitError whose message names the offending
// path, and that isEmptyRepoError correctly does not mistake it for the
// empty-repository case.
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

// TestIsEmptyRepoError covers the three shapes isEmptyRepoError must
// distinguish: the real "does not have any commits yet" stderr from a git
// invocation against a repo with no commits, a git failure for an unrelated
// reason, and an error that is not a *gitError at all.
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
		{"a non-gitError", errors.New("boom"), false},
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
