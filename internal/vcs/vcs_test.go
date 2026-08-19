package vcs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// commitEnv returns a deterministic GIT_AUTHOR_DATE/GIT_COMMITTER_DATE
// environment so fixture commits never depend on wall-clock time.
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

// runGitOrFatal runs git with args in dir, failing the test on error.
func runGitOrFatal(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// initRepo creates a real, empty git repository at dir via the git binary.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	runGitOrFatal(t, dir, "init", "-q")
	runGitOrFatal(t, dir, "config", "user.email", "test@example.com")
	runGitOrFatal(t, dir, "config", "user.name", "Test")
}

// writeAndCommit writes files (path -> content) into dir, stages them, and
// commits with msg at a deterministic author/committer date.
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

func TestOpen_EmptyRepository(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	subject, err := repo.HeadSubject()
	if err != nil {
		t.Fatalf("HeadSubject: %v", err)
	}
	if subject != "" {
		t.Fatalf("HeadSubject() = %q, want empty", subject)
	}

	subjects, err := repo.RecentSubjects(10)
	if err != nil {
		t.Fatalf("RecentSubjects: %v", err)
	}
	if subjects != nil {
		t.Fatalf("RecentSubjects() = %v, want nil", subjects)
	}

	epoch, ok, err := repo.HeadAuthorEpoch()
	if err != nil {
		t.Fatalf("HeadAuthorEpoch: %v", err)
	}
	if ok || epoch != 0 {
		t.Fatalf("HeadAuthorEpoch() = (%d, %v), want (0, false)", epoch, ok)
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

	// t.TempDir() can itself resolve through a symlink (e.g. /tmp on macOS),
	// so compare the resolved paths rather than the raw strings.
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

func TestHeadSubject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, ":sparkles: feat: first commit\n\nbody line", map[string]string{
		"a.txt": "one",
	})

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, err := repo.HeadSubject()
	if err != nil {
		t.Fatalf("HeadSubject: %v", err)
	}
	if want := ":sparkles: feat: first commit"; got != want {
		t.Fatalf("HeadSubject() = %q, want %q", got, want)
	}
}

func TestHeadAuthorEpoch(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "first", map[string]string{"a.txt": "one"})

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	epoch, ok, err := repo.HeadAuthorEpoch()
	if err != nil {
		t.Fatalf("HeadAuthorEpoch: %v", err)
	}
	if !ok {
		t.Fatal("HeadAuthorEpoch() ok = false, want true")
	}
	if want := int64(1704067200); epoch != want {
		t.Fatalf("HeadAuthorEpoch() = %d, want %d", epoch, want)
	}
}

func TestRecentSubjects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "first", map[string]string{"a.txt": "one"})
	writeAndCommit(t, dir, "second", map[string]string{"b.txt": "two"})
	writeAndCommit(t, dir, "third", map[string]string{"c.txt": "three"})

	testCases := []struct {
		name  string
		limit int
		want  []string
	}{
		{"newest first, no limit truncation", 10, []string{"third", "second", "first"}},
		{"limit caps the result", 2, []string{"third", "second"}},
		{"limit of one", 1, []string{"third"}},
		{"limit of zero returns nil", 0, nil},
		{"negative limit returns nil", -5, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo, err := Open(dir)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			got, err := repo.RecentSubjects(testCase.limit)
			if err != nil {
				t.Fatalf("RecentSubjects: %v", err)
			}
			if len(got) != len(testCase.want) {
				t.Fatalf("RecentSubjects() = %v, want %v", got, testCase.want)
			}
			for i, want := range testCase.want {
				if got[i] != want {
					t.Fatalf("RecentSubjects()[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

// TestOpen_LinkedWorktree builds a linked worktree via `git worktree add` and
// checks Open resolves it to the linked worktree's own root - git resolves
// the shared object store and commondir natively.
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

	subject, err := repo.HeadSubject()
	if err != nil {
		t.Fatalf("HeadSubject: %v", err)
	}
	if subject != "main commit" {
		t.Fatalf("HeadSubject() = %q, want %q", subject, "main commit")
	}
}

// TestCurrentBranch asserts a real git repository reports its checked-out
// branch, and that a detached HEAD reads as "" without an error - git's own
// --show-current contract, pinned here against the real binary.
func TestCurrentBranch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{
			name: "reports the checked-out branch",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeAndCommit(t, dir, "init", map[string]string{"a.txt": "a"})
				runGitOrFatal(t, dir, "checkout", "-q", "-b", "feat/thing")
			},
			want: "feat/thing",
		},
		{
			name: "an unborn branch still names itself",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				runGitOrFatal(t, dir, "checkout", "-q", "-b", "fix/unborn")
			},
			want: "fix/unborn",
		},
		{
			name: "a detached head names no branch",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeAndCommit(t, dir, "init", map[string]string{"a.txt": "a"})
				runGitOrFatal(t, dir, "checkout", "-q", "--detach")
			},
			want: "",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			initRepo(t, dir)
			testCase.setup(t, dir)

			repo, err := Open(dir)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			got, err := repo.CurrentBranch()
			if err != nil {
				t.Fatalf("CurrentBranch: %v", err)
			}
			if got != testCase.want {
				t.Errorf("CurrentBranch() = %q, want %q", got, testCase.want)
			}
		})
	}
}
