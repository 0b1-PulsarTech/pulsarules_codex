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

// stubGovernance returns block/count directly, so a test can exercise
// emitStop's control flow without running the real analyzer against a real
// repo.
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

	// The only subtest here that leaves Governance nil: it is the sole proof
	// that Dispatch still wires through to the real analyzer pipeline
	// (RunGovernanceCheck) end to end.
	t.Run("dirty tree with a finding emits the block then dedups", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitInit(t, dir)
		// An em-dash (U+2014, written via escape so this test file itself
		// carries none) trips the no-em-dash analyzer cheaply and reliably.
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
		for _, want := range []string{"Governance checks:", "no-em-dash", "go vet", "violation.go"} {
			if !strings.Contains(ctx, want) {
				t.Errorf("stop output missing %q:\n%s", want, ctx)
			}
		}
		// "Uncommitted changes:" is the (unchanged) file-list heading, not an
		// instruction, so it is excluded before checking for one.
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

func TestDispatchSubagentStop(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		dirty bool
	}{
		{name: "clean tree", dirty: false},
		{name: "dirty tree", dirty: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			gitInit(t, dir)
			if testCase.dirty {
				if err := os.WriteFile(
					filepath.Join(dir, "main.go"), []byte("package x\n"), 0o600,
				); err != nil {
					t.Fatalf("write: %v", err)
				}
			}
			disp, out := dispatchCapture(Deps{ProjectDir: dir})
			_ = disp.Dispatch("subagent-stop", newSessionPayload(t))
			// A subagent never commits, so nagging it about a dirty tree only
			// derails the work it was spawned to do.
			if strings.TrimSpace(out.String()) != "" {
				t.Errorf("subagent-stop must stay silent, got %q", out.String())
			}
		})
	}
}

// TestDispatchStop_WorktreeStatusError forces WorktreeStatus to fail while
// leaving vcs.Open able to succeed: `git rev-parse --show-toplevel` (what
// Open runs) only locates .git, while `git status --porcelain` (what
// WorktreeStatus runs) also reads the index, so a corrupted index fails the
// latter without failing the former.
func TestDispatchStop_WorktreeStatusError(t *testing.T) {
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

// TestDispatchStop_ReadStopTemplateError exercises the branch where the
// governance check finds something to report but the stop.txt template
// asset is missing, by passing a templates FS that omits it.
func TestDispatchStop_ReadStopTemplateError(t *testing.T) {
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
