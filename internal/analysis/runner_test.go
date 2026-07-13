package analysis

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestToAnalysisConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Analyzers["complexity"] = config.AnalyzerConfig{Enabled: false}

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
