package analysis

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
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
