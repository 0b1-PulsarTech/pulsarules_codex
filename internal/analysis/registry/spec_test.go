package registry

import (
	"slices"
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

func TestInternalIDs(t *testing.T) {
	t.Parallel()

	listed := InternalIDs()
	for _, spec := range specs() {
		if got := slices.Contains(listed, spec.id); got != spec.internal {
			t.Errorf("%q: listed = %v, internal flag = %v", spec.id, got, spec.internal)
		}
	}
	want := []string{"rule-injection", "output", "delegated-command"}
	slices.Sort(want)
	slices.Sort(listed)
	if !slices.Equal(listed, want) {
		t.Fatalf("InternalIDs() = %v, want %v", listed, want)
	}
}

func TestSpecs_Golden(t *testing.T) {
	t.Parallel()

	type specRow struct {
		id       string
		gate     Gate
		internal bool
	}
	testCases := []specRow{
		{"file-size", GateStatic, false},
		{"text-markers", GateStatic, false},
		{"typographic-markers", GateStatic, false},
		{"import-groups", GateStatic, false},
		{"naming", GateStatic, false},
		{"top-of-file", GateStatic, false},
		{"big-comment", GateStatic, false},
		{"simplification-path", GateStatic, false},
		{"commit-lint", GateStatic, false},
		{"commit-move-purity", GateStatic, false},
		{"go-syntax", GateStatic, false},
		{"dart-syntax", GateStatic, false},
		{"typescript-syntax", GateStatic, false},
		{"javascript-syntax", GateStatic, false},
		{"svelte-syntax", GateStatic, false},
		{"control-flow", GateStatic, false},
		{"shadowing", GateStatic, false},
		{"complexity", GateStatic, false},
		{"named-returns", GateStatic, false},
		{"test-comment", GateStatic, false},
		{"single-use-declaration", GateStatic, false},
		{"doc-comment", GateStatic, false},
		{"body-header-comment", GateStatic, false},
		{"history-narration", GateStatic, false},
		{"single-letter-name", GateStatic, false},
		{"variadic-index", GateStatic, false},
		{"too-many-results", GateStatic, false},
		{"ungrouped-interfaces", GateStatic, false},
		{"package-level-var", GateStatic, false},
		{"package-file-count", GateStatic, false},
		{"time-discipline", GateStatic, false},
		{"context-discipline", GateStatic, false},
		{"pprof-labels", GateStatic, false},
		{"branch-name", GateFullOnly, false},
		{"arch-boundary", GateArch, false},
		{"import-cycle", GateArch, false},
		{"golangci-lint", GateFullOnly, false},
		{"delegated-command", GateFullOnly, true},
		{"rule-injection", GateStatic, true},
		{"output", GateStatic, true},
	}
	table := specs()
	if len(table) != len(testCases) {
		t.Fatalf("specs() has %d rows, want %d", len(table), len(testCases))
	}
	for idx, testCase := range testCases {
		spec := table[idx]
		got := specRow{spec.id, spec.gate, spec.internal}
		if got != testCase {
			t.Errorf("specs()[%d] = %+v, want %+v", idx, got, testCase)
		}
	}
}
