package cli

import (
	"errors"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestRunGovernance_RequiresProjectDir(t *testing.T) {
	t.Setenv("CLAUDE_PROJECT_DIR", "")

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := runGovernance(inj, &version.Options{})
	if !errors.Is(err, errProjectDirRequired) {
		t.Errorf("err = %v, want %v", err, errProjectDirRequired)
	}
}

func TestGovernanceConfig(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		opts       *version.Options
		wantPreset string
		wantPath   any
	}{
		{
			name:       "no flags keeps the default preset",
			opts:       &version.Options{},
			wantPreset: config.PresetRecommended,
			wantPath:   nil,
		},
		{
			name:       "preset flag wins",
			opts:       &version.Options{Preset: config.PresetStrict},
			wantPreset: config.PresetStrict,
			wantPath:   nil,
		},
		{
			name:       "golangci config lands in the analyzer params",
			opts:       &version.Options{GolangciConfig: "build/.golangci.yml"},
			wantPreset: config.PresetRecommended,
			wantPath:   "build/.golangci.yml",
		},
		{
			name:       "preset and config path compose",
			opts:       &version.Options{Preset: config.PresetMinimal, GolangciConfig: "x.yml"},
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
			got := cfg.Analyzers["golangci-lint"].Params["config_path"]
			if got != testCase.wantPath {
				t.Errorf("config_path = %v, want %v", got, testCase.wantPath)
			}
		})
	}
}

func TestGovernanceConfig_StrictLowersFileSize(t *testing.T) {
	t.Parallel()

	cfg := governanceConfig(&version.Options{Preset: config.PresetStrict})
	if got := cfg.Analyzers["file-size"].Params["max_lines"]; got != 180 {
		t.Errorf("strict max_lines = %v, want 180", got)
	}
	if got := cfg.Analyzers["complexity"].Params["max_complexity"]; got != 10 {
		t.Errorf("strict max_complexity = %v, want 10", got)
	}
}
