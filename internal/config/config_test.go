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
