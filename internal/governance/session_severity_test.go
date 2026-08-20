package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
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
