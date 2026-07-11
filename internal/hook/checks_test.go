package hook

import (
	"os/exec"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestRunGovernanceCheck_EmptyRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	status, err := repo.WorktreeStatus()
	if err != nil {
		t.Fatalf("WorktreeStatus: %v", err)
	}

	// A repository with no commits and no changes has no findings, so the
	// pipeline must stay quiet rather than fail: an empty block and a zero
	// count, exactly.
	got, count := RunGovernanceCheck(repo, status, nil, analysis.ScopeChanged)
	if got != "" {
		t.Errorf("block = %q, want empty for an empty repo", got)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 for an empty repo", count)
	}
}
