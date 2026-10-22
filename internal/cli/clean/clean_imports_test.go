package clean

import (
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
)

func TestClean_DoesNotImportTheGovernanceOrchestrator(t *testing.T) {
	t.Parallel()

	orchestrator := reflect.TypeFor[governance.Session]().PkgPath()
	listing := reflect.TypeFor[discovery.Lister]().PkgPath()
	parsed, err := parser.ParseFile(token.NewFileSet(), "clean.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse clean.go: %v", err)
	}
	for _, spec := range parsed.Imports {
		if strings.Trim(spec.Path.Value, `"`) == orchestrator {
			t.Fatalf("clean.go imports %s; the file listing lives in %s", orchestrator, listing)
		}
	}
}
