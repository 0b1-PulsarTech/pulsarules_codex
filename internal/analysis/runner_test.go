package analysis

import (
	"testing"

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
