package hook

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func stubGovernance(block string, count int) func(vcs.Repository, vcs.Status) (string, int) {
	return func(vcs.Repository, vcs.Status) (string, int) {
		return block, count
	}
}

func TestDispatchStop(t *testing.T) {
	t.Parallel()

	t.Run("clean tree stays silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitInit(t, dir)
		disp, out := dispatchCapture(Deps{ProjectDir: dir, Governance: stubGovernance("", 0)})
		_ = disp.Dispatch("stop", newSessionPayload(t))
		if strings.TrimSpace(out.String()) != "" {
			t.Errorf("expected no output on clean tree, got %q", out.String())
		}
	})

	t.Run("dirty tree with no findings stays silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitInit(t, dir)
		if err := os.WriteFile(
			filepath.Join(dir, "clean.go"), []byte("package x\n"), 0o600,
		); err != nil {
			t.Fatalf("write: %v", err)
		}
		disp, out := dispatchCapture(Deps{ProjectDir: dir, Governance: stubGovernance("", 0)})
		_ = disp.Dispatch("stop", newSessionPayload(t))
		if strings.TrimSpace(out.String()) != "" {
			t.Errorf(
				"expected no output for a dirty tree with no governance findings, got %q",
				out.String(),
			)
		}
	})

	t.Run("dirty tree with a finding emits the block then dedups", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitInit(t, dir)
		emDash := "\u2014"
		content := "package x\n\n// bad note " + emDash + " trips no-em-dash\n"
		if err := os.WriteFile(
			filepath.Join(dir, "violation.go"),
			[]byte(content),
			0o600,
		); err != nil {
			t.Fatalf("write: %v", err)
		}
		disp, out := dispatchCapture(Deps{ProjectDir: dir})
		payload := newSessionPayload(t)

		_ = disp.Dispatch("stop", payload)
		ctx := extractContext(t, out.String())
		for _, want := range []string{
			"Governance checks:",
			"no-em-dash",
			"go vet",
			"violation.go",
		} {
			if !strings.Contains(ctx, want) {
				t.Errorf("stop output missing %q:\n%s", want, ctx)
			}
		}
		withoutHeading := strings.ReplaceAll(strings.ToLower(ctx), "uncommitted", "")
		if strings.Contains(withoutHeading, "commit") {
			t.Errorf("stop output must not instruct a commit:\n%s", ctx)
		}

		out.Reset()
		_ = disp.Dispatch("stop", payload)
		if strings.TrimSpace(out.String()) != "" {
			t.Errorf("identical change set should dedup to empty, got %q", out.String())
		}
	})
}

func TestDispatchStop_CorruptIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, ".git", "index"), []byte("not an index"), 0o600,
	); err != nil {
		t.Fatalf("corrupt index: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, out := dispatchCapture(Deps{ProjectDir: dir, Logger: logger})

	if err := disp.Dispatch("stop", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected no hook output, got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "read worktree status") {
		t.Errorf("log missing worktree status failure:\n%s", logBuf.String())
	}
}

func TestDispatchStop_MissingStopTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, "dirty.go"),
		[]byte("package x\n"),
		0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, out := dispatchCapture(Deps{
		ProjectDir: dir,
		Logger:     logger,
		Templates:  fstest.MapFS{},
		Governance: stubGovernance("\nGovernance checks:\n  - stub finding\n", 1),
	})

	if err := disp.Dispatch("stop", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected no hook output, got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "read stop.txt") {
		t.Errorf("log missing stop.txt read failure:\n%s", logBuf.String())
	}
}
