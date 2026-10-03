package analysis

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
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

type statusFailingRepo struct{ *goldenRepo }

func (statusFailingRepo) WorktreeStatus() (vcs.Status, error) {
	return vcs.Status{}, errors.New("status exploded")
}

func TestDiscover_LogsWorktreeStatusFailure(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	sess := NewSession(statusFailingRepo{&goldenRepo{root: t.TempDir()}}, "", nil, nil)
	sess.logger = slog.New(slog.NewTextHandler(&logs, nil))

	discovery := sess.Discover(ScopeFull, nil, FileSetChanged)
	if len(discovery.ChangedFiles) != 0 {
		t.Fatalf("ChangedFiles = %v, want none", discovery.ChangedFiles)
	}
	if !strings.Contains(logs.String(), "status exploded") {
		t.Fatalf("log output %q does not report the status failure", logs.String())
	}
}
