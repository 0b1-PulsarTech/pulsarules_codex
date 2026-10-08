package registry

import (
	"slices"
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

func TestLangSpec_InjectedRegistry(t *testing.T) {
	t.Parallel()

	langs := pass.NewLanguageRegistry()
	spec := langSpec("probe", func(langs *pass.LanguageRegistry) *langProbe {
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

func TestIDs_PackageLevelVarOrder(t *testing.T) {
	t.Parallel()

	ids := IDs()
	dependency := slices.Index(ids, "single-use-declaration")
	dependent := slices.Index(ids, "package-level-var")
	if dependency < 0 || dependent < 0 {
		t.Fatalf("missing ids: dependency=%d dependent=%d", dependency, dependent)
	}
	if dependency >= dependent {
		t.Fatalf("single-use-declaration at %d must precede package-level-var at %d",
			dependency, dependent)
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

func TestDelegatedFormats(t *testing.T) {
	t.Parallel()

	if !slices.Contains(DelegatedFormats(), "dart-analyze-json") {
		t.Fatalf("DelegatedFormats() = %v, want dart-analyze-json", DelegatedFormats())
	}
}
