package vcs

import (
	"strings"
	"testing"
)

// stageRename removes old.go, writes newPath with newContent, and stages
// both so the index carries a staged delete + staged add pair.
func stageRename(t *testing.T, dir, newPath, newContent string) {
	t.Helper()
	mustRemove(t, dir, "old.go")
	runGitOrFatal(t, dir, "add", "old.go")
	writeFile(t, dir, newPath, newContent)
	runGitOrFatal(t, dir, "add", newPath)
}

func TestStagedRenames_ExactMatch(t *testing.T) {
	t.Parallel()

	body := "line one\nline two\nline three\n"
	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"old.go": body})
	stageRename(t, dir, "new.go", body)

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	renames, err := repo.StagedRenames(90)
	if err != nil {
		t.Fatalf("StagedRenames: %v", err)
	}
	if len(renames) != 1 {
		t.Fatalf("StagedRenames() = %+v, want exactly one pair", renames)
	}
	got := renames[0]
	if got.OldPath != "old.go" || got.NewPath != "new.go" {
		t.Fatalf("rename = %+v, want old.go -> new.go", got)
	}
	if got.Score != 100 {
		t.Fatalf("Score = %d, want 100", got.Score)
	}
}

func TestStagedRenames_PartialMatch(t *testing.T) {
	t.Parallel()

	// 4 lines, 3 shared with the original: 3/4 = 75%.
	original := "a\nb\nc\nd\n"
	changed := "a\nb\nc\nz\n"
	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"old.go": original})
	stageRename(t, dir, "new.go", changed)

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	testCases := []struct {
		name     string
		minScore int
		wantAny  bool
	}{
		{"at threshold matches", 75, true},
		{"below threshold matches", 50, true},
		{"above the actual score excludes it", 90, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			renames, err := repo.StagedRenames(testCase.minScore)
			if err != nil {
				t.Fatalf("StagedRenames: %v", err)
			}
			if got := len(renames) > 0; got != testCase.wantAny {
				t.Fatalf("StagedRenames(%d) = %+v, want present=%v",
					testCase.minScore, renames, testCase.wantAny)
			}
			if testCase.wantAny && renames[0].Score < 70 {
				t.Fatalf("Score = %d, want roughly 75", renames[0].Score)
			}
		})
	}
}

func TestStagedRenames_UnrelatedFilesNotPaired(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{
		"old.go": "alpha\nbeta\ngamma\ndelta\n",
	})
	stageRename(t, dir, "unrelated.go", "one\ntwo\nthree\nfour\n")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	renames, err := repo.StagedRenames(50)
	if err != nil {
		t.Fatalf("StagedRenames: %v", err)
	}
	if len(renames) != 0 {
		t.Fatalf("StagedRenames() = %+v, want no pairs for unrelated content", renames)
	}
}

func TestStagedRenames_NoStagedChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	renames, err := repo.StagedRenames(50)
	if err != nil {
		t.Fatalf("StagedRenames: %v", err)
	}
	if renames != nil {
		t.Fatalf("StagedRenames() = %+v, want nil", renames)
	}
}

func TestStagedRenames_EmptyRepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	renames, err := repo.StagedRenames(50)
	if err != nil {
		t.Fatalf("StagedRenames: %v", err)
	}
	if renames != nil {
		t.Fatalf("StagedRenames() = %+v, want nil on an empty repository", renames)
	}
}

func TestStagedDiff(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})
	writeFile(t, dir, "a.go", "package a\n\nvar x = 1\n")
	runGitOrFatal(t, dir, "add", "a.go")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	diff, err := repo.StagedDiff("a.go")
	if err != nil {
		t.Fatalf("StagedDiff: %v", err)
	}
	if !strings.Contains(diff, "+var x = 1") {
		t.Fatalf("StagedDiff() = %q, want it to contain the added line", diff)
	}
}

func TestStagedRenameDiff(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	body := "line one\nline two\nline three\nline four\n"
	writeAndCommit(t, dir, "initial", map[string]string{"old.go": body})
	stageRename(t, dir, "new.go", body+"var extra = 1\n")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	diff, err := repo.StagedRenameDiff("old.go", "new.go", 1)
	if err != nil {
		t.Fatalf("StagedRenameDiff: %v", err)
	}
	if !strings.Contains(diff, "rename from old.go") {
		t.Fatalf("StagedRenameDiff() = %q, want it to show the rename pair", diff)
	}
	if !strings.Contains(diff, "+var extra = 1") {
		t.Fatalf("StagedRenameDiff() = %q, want it to contain only the added line", diff)
	}
	if strings.Contains(diff, "+line one") {
		t.Fatalf("StagedRenameDiff() = %q, want the shared lines to stay out of the delta", diff)
	}
}
