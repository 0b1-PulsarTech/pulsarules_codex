package config

import "testing"

func TestDefaults(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	if cfg.Emoji.WindowSize != 5 {
		t.Errorf("WindowSize = %d, want 5", cfg.Emoji.WindowSize)
	}
	if cfg.Emoji.SoftWindowSize != 20 {
		t.Errorf("SoftWindowSize = %d, want 20", cfg.Emoji.SoftWindowSize)
	}
	if cfg.Emoji.SuggestionCount != 7 {
		t.Errorf("SuggestionCount = %d, want 7", cfg.Emoji.SuggestionCount)
	}
	if cfg.Emoji.SoftWindowSize <= cfg.Emoji.WindowSize {
		t.Error("the advisory window must reach past the blocking one")
	}
}

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
				cfg.Analyzers[analyzerID] = AnalyzerConfig{Params: testCase.seeded}
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
