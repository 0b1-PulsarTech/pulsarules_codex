package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type langProbe struct {
	langs *pass.LanguageRegistry
}

func (probe *langProbe) ID() string                                { return "probe" }
func (probe *langProbe) Stage() finding.StageID                    { return finding.StageAST }
func (probe *langProbe) Analyze(_ *pass.Context) []finding.Finding { return nil }

func (probe *langProbe) Needs() finding.Requirements {
	return finding.Requirements{}
}

func TestLangSpec_InjectedRegistry(t *testing.T) {
	t.Parallel()

	langs := pass.NewLanguageRegistry()
	spec := langSpec(func(langs *pass.LanguageRegistry) *langProbe {
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
