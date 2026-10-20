package vcs

import (
	"slices"
	"strings"
	"testing"
)

func assertEntriesDiffer(t *testing.T, repo *Repository) {
	t.Helper()

	headMode, headID, err := repo.Entry(Head, "a.go")
	if err != nil {
		t.Fatalf("Entry(Head): %v", err)
	}
	stagedMode, stagedID, err := repo.Entry(Staged, "a.go")
	if err != nil {
		t.Fatalf("Entry(Staged): %v", err)
	}
	if headMode != "100644" || stagedMode != "100644" || headID == stagedID {
		t.Fatalf("a.go head %s %s staged %s %s, want same mode and differing ids",
			headMode, headID, stagedMode, stagedID)
	}
}

func TestEntriesAndDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{
		"a.go":        "package a\n",
		"sub/b.go":    "package b\n",
		"sub/c.go":    "package b\n",
		"sub/in/d.go": "package d\n",
		"gl[ob].go":   "package g\n",
	})
	writeFile(t, dir, "a.go", "package changed\n")
	writeFile(t, dir, "sub/e.go", "package b\n")
	runGitOrFatal(t, dir, "add", "a.go", "sub/e.go")
	runGitOrFatal(t, dir, "rm", "-q", "sub/c.go")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	type entryCase struct {
		name     string
		side     Side
		path     string
		wantMode string
		wantErr  bool
	}
	assertEntriesDiffer(t, repo)
	testCases := []entryCase{
		{
			name:     "head literal glob name",
			side:     Head,
			path:     "gl[ob].go",
			wantMode: "100644",
		},
		{
			name:     "staged literal glob name",
			side:     Staged,
			path:     "gl[ob].go",
			wantMode: "100644",
		},
		{name: "head dir entry", side: Head, path: "sub", wantMode: "040000"},
		{name: "head missing", side: Head, path: "none.go", wantErr: true},
		{name: "staged missing", side: Staged, path: "none.go", wantErr: true},
		{name: "staged deleted", side: Staged, path: "sub/c.go", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mode, id, err := repo.Entry(testCase.side, testCase.path)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if mode != testCase.wantMode || (!testCase.wantErr && id == "") {
				t.Fatalf("mode %q id %q, want mode %q and an id", mode, id, testCase.wantMode)
			}
		})
	}
}

func TestListDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{
		"a.go": "package a\n", "gl[ob].go": "package g\n",
		"sub/b.go": "package b\n", "sub/c.go": "package b\n", "sub/in/d.go": "package d\n",
	})
	writeFile(t, dir, "sub/e.go", "package b\n")
	runGitOrFatal(t, dir, "add", "sub/e.go")
	runGitOrFatal(t, dir, "rm", "-q", "sub/c.go")
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	type dirCase struct {
		name string
		side Side
		dir  string
		want []string
	}
	dirCases := []dirCase{
		{
			"head sub lists subtree names",
			Head,
			"sub",
			[]string{"sub/b.go", "sub/c.go", "sub/in"},
		},
		{"staged sub", Staged, "sub", []string{"sub/b.go", "sub/e.go"}},
		{"head root skips nested", Head, ".", []string{"a.go", "gl[ob].go", "sub"}},
		{"staged root skips nested", Staged, ".", []string{"a.go", "gl[ob].go"}},
		{"head missing dir", Head, "nope", nil},
		{"staged nested dir", Staged, "sub/in", []string{"sub/in/d.go"}},
	}
	for _, testCase := range dirCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.Dir(testCase.side, testCase.dir)
			if err != nil {
				t.Fatalf("read %s: %v", testCase.dir, err)
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestEntry_ErrorNamesTheSide(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	testCases := []struct {
		name string
		side Side
		want string
	}{
		{"head", Head, "read HEAD entry none.go:"},
		{"staged", Staged, "read staged entry none.go:"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := repo.Entry(testCase.side, "none.go")
			if err == nil || !strings.HasPrefix(err.Error(), testCase.want) {
				t.Fatalf("err = %v, want prefix %q", err, testCase.want)
			}
		})
	}
}
