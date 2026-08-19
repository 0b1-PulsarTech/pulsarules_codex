package analysis

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestToAnalysisConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetMinimal
	cfg.ApplyPreset()

	pc := toAnalysisConfig(cfg)
	if pc == nil {
		t.Fatal("expected non-nil pipeline config")
	}
	if pc.Analyzers["complexity"].Enabled != false {
		t.Error("complexity should be disabled in minimal preset")
	}
}

// TestToAnalysisConfig_ExplicitParamWinsOverProjection pins the precedence
// bug: an explicit SetParam call on commit-move-purity's severity must
// survive the config-to-param projection that runs after it, not be
// silently overwritten by config.MovePurityConfig's own default.
func TestToAnalysisConfig_ExplicitParamWinsOverProjection(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	if cfg.MovePurity.Severity == "" {
		t.Fatal("test assumes MovePurity.Severity has a default")
	}
	cfg.SetParam(movepurity.AnalyzerID, movepurity.ParamMinSimilarity, 99)
	cfg.SetParam(movepurity.AnalyzerID, "severity", "error")

	pc := toAnalysisConfig(cfg)
	params := pc.Analyzers[movepurity.AnalyzerID].Params
	if got := params["severity"]; got != "error" {
		t.Fatalf("severity = %v, want explicit \"error\" to survive projection", got)
	}
	if got := params[movepurity.ParamMinSimilarity]; got != 99 {
		t.Fatalf("min_similarity = %v, want explicit 99 to survive projection", got)
	}
}
