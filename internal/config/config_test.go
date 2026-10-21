package config

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

func isEnabled(cfg *GovernanceConfig, analyzerID string) bool {
	ac, ok := cfg.Analyzers[analyzerID]
	if !ok {
		return true
	}
	return ac.Enabled
}

func TestSetParam(t *testing.T) {
	t.Parallel()

	const analyzerID, key, value = "golangci-lint", "config_path", "./golangci.yml"
	testCases := []struct {
		name   string
		seeded map[string]any
	}{
		{name: "creates the entry when the analyzer has none"},
		{
			name:   "overwrites an entry without dropping its other params",
			seeded: map[string]any{"timeout": 30},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := Defaults()
			if testCase.seeded != nil {
				cfg.Analyzers[analyzerID] = pass.AnalyzerConfig{Params: testCase.seeded}
			}
			cfg.SetParam(analyzerID, key, value)

			if got := cfg.Analyzers[analyzerID].Params[key]; got != value {
				t.Errorf("Param(%q) = %v, want %v", key, got, value)
			}
			for otherKey, want := range testCase.seeded {
				if got := cfg.Analyzers[analyzerID].Params[otherKey]; got != want {
					t.Errorf("Param(%q) = %v, want %v", otherKey, got, want)
				}
			}
		})
	}
}

func effectiveSeverity(cfg *GovernanceConfig, analyzerID string, from finding.Severity) string {
	probe := []finding.Finding{{AnalyzerID: analyzerID, Severity: from}}
	return overrides.Apply(probe, analyzerID, cfg.SeverityRules)[0].Severity.String()
}

func TestSetSeverity(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.SetSeverity("typographic-markers", "error")
	cfg.SetSeverity("doc-comment", "info")
	cfg.SetSeverity("typographic-markers", "warning")

	testCases := []struct {
		analyzerID string
		want       string
	}{
		{"typographic-markers", "warning"},
		{"doc-comment", "info"},
		{"naming", "error"},
	}
	for _, testCase := range testCases {
		got := effectiveSeverity(cfg, testCase.analyzerID, finding.SeverityError)
		if got != testCase.want {
			t.Errorf("severity(%s) = %q, want %q", testCase.analyzerID, got, testCase.want)
		}
	}
	for _, rule := range cfg.SeverityRules {
		if !rule.Files.IsOpen() {
			t.Errorf("severity rule is scoped to files: %+v", rule)
		}
	}
}

func TestAnalysis(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetMinimal
	cfg.ApplyPreset()

	cfg.Analyzers["kept-on"] = pass.AnalyzerConfig{Enabled: true}

	pc := cfg.Analysis()
	if pc == nil {
		t.Fatal("expected non-nil pipeline config")
	}
	testCases := []struct {
		id      string
		enabled bool
	}{
		{"complexity", false},
		{"kept-on", true},
	}
	for _, testCase := range testCases {
		got, found := pc.Analyzers[testCase.id]
		if !found || got.Enabled != testCase.enabled {
			t.Errorf("%s = %+v (found %v), want Enabled %v kept", testCase.id, got, found,
				testCase.enabled)
		}
	}
}

func TestAnalysis_ParamsPassThrough(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.SetParam(movepurity.ID, movepurity.ParamMinSimilarity, 99)
	cfg.SetParam(message.ID, message.ParamEmojiHardWindow, 3)

	pc := cfg.Analysis()
	if got := pc.Analyzers[movepurity.ID].Params[movepurity.ParamMinSimilarity]; got != 99 {
		t.Fatalf("min_similarity = %v, want the explicit 99", got)
	}
	if got := pc.Analyzers[message.ID].Params[message.ParamEmojiHardWindow]; got != 3 {
		t.Fatalf("emoji_hard_window = %v, want the explicit 3", got)
	}
}

func TestAnalysis_DefaultsCarryNoAnalyzerParams(t *testing.T) {
	t.Parallel()

	pc := Defaults().Analysis()
	for _, id := range []string{message.ID, movepurity.ID} {
		if entry, found := pc.Analyzers[id]; found {
			t.Errorf("%s = %+v, want no entry: the analyzer owns its defaults", id, entry)
		}
	}
}

func TestAnalysis_AnalyzersAreNotShared(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.SetParam("alpha", "key", 1)
	got := cfg.Analysis()

	got.Analyzers["beta"] = pass.AnalyzerConfig{Enabled: true}
	got.Analyzers["alpha"] = pass.AnalyzerConfig{}
	if _, leaked := cfg.Analyzers["beta"]; leaked {
		t.Error("an entry added to the analysis config appeared on the governance config")
	}
	if !cfg.Analyzers["alpha"].Enabled {
		t.Error("an entry replaced on the analysis config changed the governance config")
	}

	cfg.Analyzers["gamma"] = pass.AnalyzerConfig{Enabled: true}
	if _, leaked := got.Analyzers["gamma"]; leaked {
		t.Error("an entry added to the governance config appeared on the analysis config")
	}
}

func TestAnalysisProjectFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(cfg *GovernanceConfig)
		check func(t *testing.T, cfg *GovernanceConfig, got *pass.AnalysisConfig)
	}{
		{"tests and delegated", setTestsAndDelegated, checkTestsAndDelegated},
		{"severity rules precede file rules", setSeverityRules, checkSeverityRules},
		{"disabled delegated is dropped", setDisabledDelegated, checkDisabledDelegated},
		{"no project file", func(*GovernanceConfig) {}, checkNoProjectFields},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := Defaults()
			testCase.setup(cfg)
			testCase.check(t, cfg, cfg.Analysis())
		})
	}
}

func setTestsAndDelegated(cfg *GovernanceConfig) {
	cfg.IncludeGenerated = true
	cfg.TestGlobs = []string{"web/test/**"}
	cfg.Delegated = []pass.DelegatedCommand{{ID: "dart-analyze", Run: []string{"dart"}}}
}

func checkTestsAndDelegated(t *testing.T, cfg *GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if !got.IncludeGenerated {
		t.Error("IncludeGenerated = false, want true")
	}
	if !reflect.DeepEqual(got.TestGlobs, cfg.TestGlobs) {
		t.Errorf("TestGlobs = %v", got.TestGlobs)
	}
	if !reflect.DeepEqual(got.Delegated, cfg.Delegated) {
		t.Errorf("Delegated = %v", got.Delegated)
	}
}

func setSeverityRules(cfg *GovernanceConfig) {
	cfg.SetSeverity("naming", "info")
	cfg.SetSeverity("big-comment", "error")
	cfg.Overrides = []pass.Override{{Files: glob.CompileSet([]string{"a/**"})}}
}

func checkSeverityRules(t *testing.T, _ *GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if len(got.Overrides) != 3 {
		t.Fatalf("Overrides = %+v, want 2 severity rules then the file rule", got.Overrides)
	}
	if _, first := got.Overrides[0].Analyzers["naming"]; !first {
		t.Errorf("severity rules lost their order: %+v", got.Overrides)
	}
	if !got.Overrides[2].Covers("a/x") || got.Overrides[2].Covers("b/x") {
		t.Errorf("file rule must come after the severity rules: %+v", got.Overrides)
	}
}

func checkNoProjectFields(t *testing.T, _ *GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if len(got.Overrides) != 0 || len(got.TestGlobs) != 0 || len(got.Delegated) != 0 {
		t.Fatalf("project fields set without a project file: %+v", got)
	}
}

func setDisabledDelegated(cfg *GovernanceConfig) {
	cfg.Delegated = []pass.DelegatedCommand{
		{ID: "off", Run: []string{"a"}},
		{ID: "on"},
		{ID: "unlisted"},
	}
	cfg.Analyzers = map[string]pass.AnalyzerConfig{
		"off": {Enabled: false},
		"on":  {Enabled: true},
	}
}

func commandIDs(entries []pass.DelegatedCommand) []string {
	ids := make([]string, len(entries))
	for idx, entry := range entries {
		ids[idx] = entry.ID
	}
	return ids
}

func checkDisabledDelegated(
	t *testing.T, cfg *GovernanceConfig, got *pass.AnalysisConfig,
) {
	t.Helper()
	if ids := commandIDs(got.Delegated); !slices.Equal(ids, []string{"on", "unlisted"}) {
		t.Errorf("Delegated ids = %v, want [on unlisted]", ids)
	}
	wantSource := []string{"off", "on", "unlisted"}
	if ids := commandIDs(cfg.Delegated); !slices.Equal(ids, wantSource) {
		t.Errorf("source Delegated ids = %v, want %v", ids, wantSource)
	}
}
