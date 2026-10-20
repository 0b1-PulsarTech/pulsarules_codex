package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func mergeConflict(t *testing.T, dir string) {
	t.Helper()
	writeAndCommit(t, dir, "base", map[string]string{"c.go": "base\n"})
	runGitOrFatal(t, dir, "checkout", "-q", "-b", "other")
	writeAndCommit(t, dir, "other", map[string]string{"c.go": "other\n"})
	runGitOrFatal(t, dir, "checkout", "-q", "-")
	writeAndCommit(t, dir, "main", map[string]string{"c.go": "main\n"})
	merge := exec.CommandContext(
		context.Background(), "git", "-C", dir, "merge", "--no-ff", "--no-edit", "other",
	)
	_ = merge.Run()
	unmerged, err := exec.CommandContext(
		context.Background(), "git", "-C", dir, "ls-files", "-u",
	).Output()
	if err != nil || len(unmerged) == 0 {
		t.Fatalf("merge: expected unmerged entries, got %q (%v)", unmerged, err)
	}
}

func setupIgnoredDir(t *testing.T, dir string) {
	t.Helper()
	writeAndCommit(t, dir, "init", map[string]string{
		".gitignore": "docs/local/\n",
		"a.go":       "package a\n",
	})
	writeFile(t, dir, "docs/local/PLAN.md", "x")
	writeFile(t, dir, "docs/keep.md", "x")
}

func setupDeleted(t *testing.T, dir string) {
	t.Helper()
	writeAndCommit(t, dir, "init", map[string]string{"a.go": "x", "gone.go": "x"})
	if err := os.Remove(filepath.Join(dir, "gone.go")); err != nil {
		t.Fatalf("remove: %v", err)
	}
}

func setupNestedIgnore(t *testing.T, dir string) {
	t.Helper()
	writeAndCommit(t, dir, "init", map[string]string{
		"sub/.gitignore": "secret.txt\n",
		"sub/a.go":       "x",
	})
	writeFile(t, dir, "sub/secret.txt", "x")
	writeFile(t, dir, "secret.txt", "x")
}

func setupGitlink(t *testing.T, dir string) {
	t.Helper()
	writeAndCommit(t, dir, "init", map[string]string{"a.go": "x"})
	if err := os.MkdirAll(filepath.Join(dir, "sm"), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	runGitOrFatal(t, dir, "update-index", "--add", "--cacheinfo",
		"160000,1111111111111111111111111111111111111111,sm")
}

func setupUnicode(t *testing.T, dir string) {
	t.Helper()
	writeAndCommit(t, dir, "init", map[string]string{"my dir/a b.go": "x"})
	writeFile(t, dir, "caf\u00e9/\u65e5\u672c.go", "x")
}

type listFilesCase struct {
	name    string
	noRepo  bool
	setup   func(t *testing.T, dir string)
	want    []string
	wantErr bool
}

func listFilesCases() []listFilesCase {
	return []listFilesCase{
		{
			name:  "ignored directory is excluded",
			setup: setupIgnoredDir,
			want:  []string{".gitignore", "a.go", "docs/keep.md"},
		},
		{
			name: "untracked file is included",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeAndCommit(t, dir, "init", map[string]string{"a.go": "package a\n"})
				writeFile(t, dir, "b/new.go", "package b\n")
			},
			want: []string{"a.go", "b/new.go"},
		},
		{
			name:  "deleted tracked file is dropped",
			setup: setupDeleted,
			want:  []string{"a.go"},
		},
		{
			name:  "nested gitignore applies",
			setup: setupNestedIgnore,
			want:  []string{"secret.txt", "sub/.gitignore", "sub/a.go"},
		},
		{
			name:  "unmerged path is listed once",
			setup: mergeConflict,
			want:  []string{"c.go"},
		},
		{
			name:  "gitlink is dropped",
			setup: setupGitlink,
			want:  []string{"a.go"},
		},
		{
			name:  "spaces and unicode survive",
			setup: setupUnicode,
			want:  []string{"caf\u00e9/\u65e5\u672c.go", "my dir/a b.go"},
		},
		{
			name:    "outside a repository fails",
			noRepo:  true,
			setup:   func(t *testing.T, dir string) { t.Helper() },
			wantErr: true,
		},
	}
}

func TestRepository_ListFiles(t *testing.T) {
	t.Parallel()

	testCases := listFilesCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if !testCase.noRepo {
				initRepo(t, dir)
				runGitOrFatal(t, dir, "config", "core.excludesFile", os.DevNull)
			}
			testCase.setup(t, dir)

			got, err := (&Repository{root: dir}).ListFiles()
			if (err != nil) != testCase.wantErr {
				t.Fatalf("ListFiles err = %v, wantErr %v", err, testCase.wantErr)
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("ListFiles = %v, want %v", got, testCase.want)
			}
		})
	}
}
