package registry

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
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
		if got := slices.Contains(listed, spec.ID); got == spec.internal {
			t.Errorf("%q: listed = %v, internal flag = %v", spec.ID, got, spec.internal)
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
		{"history-narration", GateStatic, false},
		{"branch-name", GateFullOnly, false},
		{"golangci-lint", GateFullOnly, false},
		{"delegated-command", GateFullOnly, true},
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
		{"single-letter-name", GateStatic, false},
		{"variadic-index", GateStatic, false},
		{"too-many-results", GateStatic, false},
		{"ungrouped-interfaces", GateStatic, false},
		{"package-level-var", GateStatic, false},
		{"package-file-count", GateStatic, false},
		{"time-discipline", GateStatic, false},
		{"context-discipline", GateStatic, false},
		{"pprof-labels", GateStatic, false},
		{"arch-boundary", GateArch, false},
		{"import-cycle", GateArch, false},
		{"rule-injection", GateStatic, true},
		{"output", GateStatic, true},
	}
	table := specs()
	if len(table) != len(testCases) {
		t.Fatalf("specs() has %d rows, want %d", len(table), len(testCases))
	}
	for idx, testCase := range testCases {
		spec := table[idx]
		got := specRow{spec.ID, spec.gate, spec.internal}
		if got != testCase {
			t.Errorf("specs()[%d] = %+v, want %+v", idx, got, testCase)
		}
	}
}

func TestParams_Golden(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"file-size":           {"max_lines", "error_lines", "extensions"},
		"big-comment":         {"max_lines", "extensions"},
		"text-markers":        {"extensions"},
		"typographic-markers": {"extensions"},
		"top-of-file":         {"extensions"},
		"simplification-path": {"extensions"},
		"history-narration":   {"extensions"},
		"package-file-count":  {"max_files", "exempt", "extensions"},
		"complexity":          {"max_complexity", "max_func_lines", "max_params"},
		"single-letter-name":  {"exported_only", "allowed_names"},
		"too-many-results":    {"max_results"},
		"body-header-comment": {"min_lines"},
		"branch-name":         {"extra_types"},
		"golangci-lint":       {"config_path"},
		"commit-lint":         {"emoji_hard_window", "emoji_soft_window", "emoji_suggestions"},
		"commit-move-purity":  {"min_similarity"},
	}
	if got := Params(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Params() = %v, want %v", got, want)
	}
}

func TestBuild_FailureSurfaces(t *testing.T) {
	t.Parallel()

	failing := errors.New("catalog unavailable")
	broken := viaDeps(pass.Descriptor{ID: "broken"}, GateStatic, nil)
	broken.build = func(Deps) (pass.Analyzer, error) { return nil, failing }
	table := []entry{
		viaDeps(pass.Descriptor{ID: "first"}, GateStatic, func(Deps) pass.Analyzer { return nil }),
		broken,
		viaDeps(pass.Descriptor{ID: "late"}, GateFullOnly, func(Deps) pass.Analyzer { return nil }),
	}

	built, err := build(table, Deps{}, []Gate{GateStatic})

	if !errors.Is(err, failing) || !strings.Contains(err.Error(), "build broken: ") {
		t.Fatalf("build error = %v, want it to wrap %v naming broken", err, failing)
	}
	if len(built) != 1 {
		t.Fatalf("built %d analyzers, want the one that did not fail", len(built))
	}
}
