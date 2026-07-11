package cli

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestGovernanceConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       *cliopts.Options
		wantPreset string
		wantPath   any
	}{
		{
			name:       "no flags keeps the default preset",
			opts:       &cliopts.Options{},
			wantPreset: config.PresetRecommended,
			wantPath:   nil,
		},
		{
			name:       "preset flag wins",
			opts:       &cliopts.Options{Preset: config.PresetStrict},
			wantPreset: config.PresetStrict,
			wantPath:   nil,
		},
		{
			name:       "golangci config lands in the analyzer params",
			opts:       &cliopts.Options{GolangciConfig: "build/.golangci.yml"},
			wantPreset: config.PresetRecommended,
			wantPath:   "build/.golangci.yml",
		},
		{
			name:       "preset and config path compose",
			opts:       &cliopts.Options{Preset: config.PresetMinimal, GolangciConfig: "x.yml"},
			wantPreset: config.PresetMinimal,
			wantPath:   "x.yml",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := governanceConfig(testCase.opts)
			if cfg.Preset != testCase.wantPreset {
				t.Errorf("Preset = %q, want %q", cfg.Preset, testCase.wantPreset)
			}
			got := cfg.Param("golangci-lint", "config_path", nil)
			if got != testCase.wantPath {
				t.Errorf("config_path = %v, want %v", got, testCase.wantPath)
			}
		})
	}
}

// TestGovernanceConfig_StrictLowersFileSize pins the one preset override that
// still differs from the analyzer's own default, so deleting it as
// "redundant" fails here instead of silently loosening --preset strict.
func TestGovernanceConfig_StrictLowersFileSize(t *testing.T) {
	t.Parallel()

	cfg := governanceConfig(&cliopts.Options{Preset: config.PresetStrict})
	if got := cfg.Param("file-size", "max_lines", 999); got != 180 {
		t.Errorf("strict max_lines = %v, want 180", got)
	}
	if got := cfg.Param("complexity", "max_complexity", 999); got != 10 {
		t.Errorf("strict max_complexity = %v, want 10", got)
	}
}
