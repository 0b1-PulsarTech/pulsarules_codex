package vcs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCommonDir asserts CommonDir resolves the shared git dir for a normal
// checkout and for a linked worktree - where dir/.git is a file rather than a
// directory - and reports an error outside a repository.
func TestCommonDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		setup   func(t *testing.T, root string) (dir, want string)
		wantErr bool
	}{
		{
			name: "normal checkout resolves its own git dir",
			setup: func(t *testing.T, root string) (string, string) {
				t.Helper()
				initRepo(t, root)
				return root, filepath.Join(root, ".git")
			},
		},
		{
			name: "linked worktree resolves the main repository git dir",
			setup: func(t *testing.T, root string) (string, string) {
				t.Helper()
				mainDir := filepath.Join(root, "main")
				if err := os.MkdirAll(mainDir, 0o750); err != nil {
					t.Fatalf("mkdir main: %v", err)
				}
				initRepo(t, mainDir)
				writeAndCommit(t, mainDir, "init", map[string]string{"a.txt": "a"})
				worktree := filepath.Join(root, "wt")
				runGitOrFatal(t, mainDir, "worktree", "add", "-q", worktree, "-b", "probe")
				// The pointer file is the whole reason CommonDir exists.
				entry, statErr := os.Lstat(filepath.Join(worktree, ".git"))
				if statErr != nil {
					t.Fatalf("lstat worktree .git: %v", statErr)
				}
				if entry.IsDir() {
					t.Fatal("worktree .git is a directory, expected a gitdir pointer file")
				}
				return worktree, filepath.Join(mainDir, ".git")
			},
		},
		{
			name: "outside a repository reports an error",
			setup: func(t *testing.T, root string) (string, string) {
				t.Helper()
				return root, ""
			},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir, want := testCase.setup(t, t.TempDir())
			got, err := CommonDir(dir)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("CommonDir(%q) err = nil, want an error", dir)
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

// resolved expands every symlink in path so a comparison survives a temp dir
// that git reports in its resolved form (/private/var on darwin) while the
// test holds the unresolved one.
func resolved(t *testing.T, path string) string {
	t.Helper()
	out, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("resolve %q: %v", path, err)
	}
	return out
}
