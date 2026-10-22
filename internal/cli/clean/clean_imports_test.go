package clean

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestClean_DoesNotImportTheAnalysisOrchestrator(t *testing.T) {
	t.Parallel()

	const orchestrator = "github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	parsed, err := parser.ParseFile(token.NewFileSet(), "clean.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse clean.go: %v", err)
	}
	for _, spec := range parsed.Imports {
		if strings.Trim(spec.Path.Value, `"`) == orchestrator {
			t.Fatalf(
				"clean.go imports %s; the file listing lives in analysis/discovery",
				orchestrator,
			)
		}
	}
}
