package registry

import (
	"reflect"
	"slices"
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

func TestBuild_GatesAreOrdered(t *testing.T) {
	t.Parallel()

	spec := func(id string, gate Gate) entry {
		return viaDeps(pass.Descriptor{ID: id}, gate, func(Deps) pass.Analyzer { return nil })
	}
	table := []entry{
		spec("full", GateFullOnly), spec("static", GateStatic), spec("arch", GateArch),
	}
	testCases := []struct {
		name    string
		maxGate Gate
		want    []string
	}{
		{name: "static", maxGate: GateStatic, want: []string{"static"}},
		{name: "arch", maxGate: GateArch, want: []string{"static", "arch"}},
		{name: "full", maxGate: GateFullOnly, want: []string{"full", "static", "arch"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			built := build(table, Deps{}, testCase.maxGate)
			got := make([]string, len(built))
			for idx, analyzer := range built {
				got[idx] = analyzer.ID
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("built %v, want %v", got, testCase.want)
			}
		})
	}
}
