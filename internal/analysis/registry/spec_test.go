package registry

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type langProbe struct {
	langs *core.LanguageRegistry
}

func (probe *langProbe) ID() string                                     { return "probe" }
func (probe *langProbe) Stage() core.StageID                            { return core.StageAST }
func (probe *langProbe) Analyze(_ *core.AnalysisContext) []core.Finding { return nil }

func TestLangSpec_InjectedRegistry(t *testing.T) {
	t.Parallel()

	langs := core.NewLanguageRegistry()
	spec := langSpec("probe", func(langs *core.LanguageRegistry) *langProbe {
		return &langProbe{langs: langs}
	})
	built, ok := spec.build(nil, langs, nil).(*langProbe)
	if !ok {
		t.Fatalf("build did not return the constructed analyzer")
	}
	if built.langs != langs {
		t.Fatalf("constructor got %p, want the injected %p", built.langs, langs)
	}
}

func TestIDs_NoDuplicates(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for _, id := range IDs() {
		if seen[id] {
			t.Errorf("duplicate analyzer id %q", id)
		}
		seen[id] = true
	}
}
