package registry

import (
	"slices"
	"testing"
)

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

func TestConfigurableIDs(t *testing.T) {
	t.Parallel()

	listed := ConfigurableIDs()
	for _, spec := range specs() {
		if got := slices.Contains(listed, spec.id); got == spec.internal {
			t.Errorf("%q: listed = %v, internal flag = %v", spec.id, got, spec.internal)
		}
	}
	for _, hidden := range []string{"rule-injection", "output", "delegated-command"} {
		if slices.Contains(listed, hidden) {
			t.Fatalf("ConfigurableIDs() = %v, want %q hidden", listed, hidden)
		}
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
