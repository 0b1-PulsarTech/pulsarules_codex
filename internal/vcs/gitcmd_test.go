package vcs

import (
	"strings"
	"testing"
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
}
