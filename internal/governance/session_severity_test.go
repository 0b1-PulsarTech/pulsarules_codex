package governance

import (
	"reflect"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestToAnalysisConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetMinimal
	cfg.ApplyPreset()

	cfg.Analyzers["kept-on"] = config.AnalyzerConfig{Enabled: true}

	pc := toAnalysisConfig(cfg)
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

func TestToAnalysisConfig_ExplicitSetParam(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	if cfg.MovePurity.Severity == "" {
		t.Fatal("test assumes MovePurity.Severity has a default")
	}
	cfg.SetParam(movepurity.ID, movepurity.ParamMinSimilarity, 99)
	cfg.SetParam(movepurity.ID, "severity", "error")

	pc := toAnalysisConfig(cfg)
	params := pc.Analyzers[movepurity.ID].Params
	if got := params["severity"]; got != "error" {
		t.Fatalf("severity = %v, want explicit \"error\" to survive projection", got)
	}
	if got := params[movepurity.ParamMinSimilarity]; got != 99 {
		t.Fatalf("min_similarity = %v, want explicit 99 to survive projection", got)
	}
}

func TestToAnalysisConfigProjectFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		setup func(cfg *config.GovernanceConfig)
		check func(t *testing.T, cfg *config.GovernanceConfig, got *pass.AnalysisConfig)
	}{
		{"tests and delegated", setTestsAndDelegated, checkTestsAndDelegated},
		{"severity rules precede file rules", setSeverityRules, checkSeverityRules},
		{"no project file", func(*config.GovernanceConfig) {}, checkNoProjectFields},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Defaults()
			testCase.setup(cfg)
			testCase.check(t, cfg, toAnalysisConfig(cfg))
		})
	}
}

func setTestsAndDelegated(cfg *config.GovernanceConfig) {
	cfg.TestGlobs = []string{"web/test/**"}
	cfg.Delegated = []pass.DelegatedCommand{{ID: "dart-analyze", Run: []string{"dart"}}}
}

func checkTestsAndDelegated(t *testing.T, cfg *config.GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if !reflect.DeepEqual(got.TestGlobs, cfg.TestGlobs) {
		t.Errorf("TestGlobs = %v", got.TestGlobs)
	}
	if !reflect.DeepEqual(got.Delegated, cfg.Delegated) {
		t.Errorf("Delegated = %v", got.Delegated)
	}
}

func setSeverityRules(cfg *config.GovernanceConfig) {
	cfg.Severities = map[string]string{"naming": "info", "big-comment": "error"}
	cfg.Overrides = []pass.Override{{Files: []string{"a/**"}}}
}

func checkSeverityRules(t *testing.T, _ *config.GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if len(got.Overrides) != 3 {
		t.Fatalf("Overrides = %+v, want 2 severity rules then the file rule", got.Overrides)
	}
	if _, first := got.Overrides[0].Analyzers["big-comment"]; !first {
		t.Errorf("severity rules not sorted by analyzer ID: %+v", got.Overrides)
	}
	if got.Overrides[2].Files[0] != "a/**" {
		t.Errorf("file rule must come after the severity rules: %+v", got.Overrides)
	}
}

func checkNoProjectFields(t *testing.T, _ *config.GovernanceConfig, got *pass.AnalysisConfig) {
	t.Helper()
	if len(got.Overrides) != 0 || len(got.TestGlobs) != 0 || len(got.Delegated) != 0 {
		t.Fatalf("project fields set without a project file: %+v", got)
	}
}
