package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestWrite_MatchesSessionStart(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	want, err := contract.Session(templates)
	if err != nil {
		t.Fatalf("contract.Session: %v", err)
	}

	projectDir := t.TempDir()
	wrote, err := Write(templates, projectDir, idx, []string{"go-style"})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if wrote == ownedfile.Kept {
		t.Fatalf("outcome = %v, want Wrote for a fresh project dir", wrote)
	}

	//nolint:gosec // temp dir.
	raw, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !strings.Contains(string(raw), want) {
		t.Errorf(
			"AGENTS.md does not contain the contract verbatim:\nwant substring=%q\ngot=%s",
			want,
			raw,
		)
	}
}
