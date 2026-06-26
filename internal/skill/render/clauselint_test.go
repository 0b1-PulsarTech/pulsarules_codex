package render

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestBodyDefines(t *testing.T) {
	t.Parallel()

	defs, err := bodyDefines(`pre {{define "must"}}x{{end}}{{define "forbidden"}}y{{end}}`)
	if err != nil {
		t.Fatalf("bodyDefines: %v", err)
	}
	for _, want := range []string{"must", "forbidden"} {
		if !slices.Contains(defs, want) {
			t.Errorf("defines %v missing %q", defs, want)
		}
	}
	if _, err := bodyDefines("{{define bad"); err == nil {
		t.Error("expected a parse error for malformed template")
	}
}

// TestLintSections asserts the embedded index defines only canonical sections.
func TestLintSections(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := LintSections(idx); len(problems) != 0 {
		t.Errorf("embedded index should lint clean, got %v", problems)
	}
}
