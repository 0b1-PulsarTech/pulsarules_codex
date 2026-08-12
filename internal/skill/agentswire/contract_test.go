package agentswire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestWriteAgents_ContractIsVerbatim is the drift test for the third
// consumer: it renders against the repo's real embedded templates (not
// fakeTemplates) and asserts AGENTS.md contains contract.Session's text
// unchanged - the same string the SessionStart hook emits - so AGENTS.md
// cannot silently reword or drop a clause the hooks still state.
func TestWriteAgents_ContractIsVerbatim(t *testing.T) {
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
	wrote, err := WriteAgents(templates, projectDir, idx, []string{"go-style"})
	if err != nil {
		t.Fatalf("WriteAgents: %v", err)
	}
	if !wrote {
		t.Fatal("wrote = false, want true for a fresh project dir")
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
