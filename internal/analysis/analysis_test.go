package analysis

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

// gitInitRepo initializes an empty git repository at dir via the git binary,
// configuring a deterministic author identity for any commits tests make.
func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email", "test@test.com").
		Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name", "Test").
		Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}
}

// commitAllowEmpty creates an empty commit with msg in dir, which must
// already be a git repository (see gitInitRepo).
func commitAllowEmpty(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.CommandContext(
		t.Context(),
		"git",
		"-C",
		dir,
		"commit",
		"-q",
		"--allow-empty",
		"-m",
		msg,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

// commitAllowEmptyAt creates an empty commit with msg in dir at a
// deterministic author/committer date (RFC 3339, e.g.
// "2024-01-01T00:00:00Z"), for tests asserting on HEAD's own author date.
func commitAllowEmptyAt(t *testing.T, dir, msg, authorDate string) {
	t.Helper()
	cmd := exec.CommandContext(
		t.Context(),
		"git",
		"-C",
		dir,
		"commit",
		"-q",
		"--allow-empty",
		"-m",
		msg,
	)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+authorDate, "GIT_COMMITTER_DATE="+authorDate)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}
