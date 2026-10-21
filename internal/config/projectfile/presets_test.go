package projectfile

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestApplyPreset_RecommendedDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetRecommended
	cfg.ApplyPreset()

	if isEnabled(cfg, "complexity") != true {
		t.Error("complexity should be enabled in recommended")
	}
}

func TestApplyPreset_Minimal(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetMinimal
	cfg.ApplyPreset()

	if isEnabled(cfg, "complexity") {
		t.Error("complexity should be disabled in minimal preset")
	}
	if isEnabled(cfg, "arch-boundary") {
		t.Error("arch-boundary should be disabled in minimal preset")
	}
	if !isEnabled(cfg, "file-size") {
		t.Error("file-size should be enabled in minimal preset")
	}
	if !isEnabled(cfg, "naming") {
		t.Error("naming should be enabled in minimal preset")
	}
	if isEnabled(cfg, "context-discipline") {
		t.Error("context-discipline should be disabled in minimal preset")
	}
	if !isEnabled(cfg, "pprof-labels") {
		t.Error("pprof-labels should stay enabled in minimal preset: it is a security check")
	}
	if isEnabled(cfg, "test-comment") {
		t.Error("test-comment should be disabled in minimal preset")
	}
	if isEnabled(cfg, "single-use-declaration") {
		t.Error("single-use-declaration should be disabled in minimal preset")
	}
}

func TestApplyPreset_Strict(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetStrict
	cfg.ApplyPreset()

	if !isEnabled(cfg, "file-size") {
		t.Error("file-size should be enabled in strict preset")
	}
	if !isEnabled(cfg, "complexity") {
		t.Error("complexity should be enabled in strict preset")
	}
}

func TestApplyPreset_UnknownIsNoOp(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = "unknown"
	cfg.ApplyPreset()

	if !isEnabled(cfg, "complexity") {
		t.Error("complexity should still be enabled after unknown preset")
	}
}

func TestApplyPreset_StrictOverridesMaxLinesParam(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetStrict
	cfg.ApplyPreset()

	maxLines := cfg.Analyzers["file-size"].Params["max_lines"]
	if maxLines != 180 {
		t.Errorf("expected max_lines=180 in strict preset, got %v", maxLines)
	}
}

func TestValidPreset(t *testing.T) {
	t.Parallel()

	if !config.ValidPreset(config.PresetRecommended) {
		t.Error("recommended should be valid")
	}
	if !config.ValidPreset(config.PresetStrict) {
		t.Error("strict should be valid")
	}
	if !config.ValidPreset(config.PresetMinimal) {
		t.Error("minimal should be valid")
	}
	if config.ValidPreset("unknown") {
		t.Error("unknown should not be valid")
	}
}

func TestPresets(t *testing.T) {
	t.Parallel()

	const wantPresetCount = 3

	list := config.Presets()
	if len(list) != wantPresetCount {
		t.Fatalf("expected %d presets, got %d: %v", wantPresetCount, len(list), list)
	}
	if list[0] != config.PresetRecommended {
		t.Errorf("first preset should be recommended, got %q", list[0])
	}
}

func TestApplyPreset_MinimalDisablesNewAnalyzers(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"doc-comment",
		"body-header-comment",
		"history-narration",
		"variadic-index",
		"too-many-results",
		"ungrouped-interfaces",
		"package-level-var",
		"package-file-count",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			t.Parallel()

			cfg := config.Defaults()
			cfg.Preset = config.PresetMinimal
			cfg.ApplyPreset()

			if isEnabled(cfg, testCase) {
				t.Errorf("%s should be disabled in minimal preset", testCase)
			}
		})
	}
}

func TestApplyPreset_MinimalSingleLetters(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.Preset = config.PresetMinimal
	cfg.ApplyPreset()

	got := cfg.Analyzers["single-letter-name"]
	if !got.Enabled || got.Params["exported_only"] != true {
		t.Fatalf("single-letter-name = %+v, want enabled with exported_only", got)
	}
}

func TestPresetTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		preset     string
		wantValid  bool
		wantNilMap bool
	}{
		{
			name: "recommended lists no override", preset: config.PresetRecommended,
			wantValid: true, wantNilMap: true,
		},
		{name: "strict lists overrides", preset: config.PresetStrict, wantValid: true},
		{name: "minimal lists overrides", preset: config.PresetMinimal, wantValid: true},
		{name: "empty is not a preset", preset: ""},
		{name: "case matters", preset: "Strict"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := config.ValidPreset(testCase.preset); got != testCase.wantValid {
				t.Errorf("ValidPreset(%q) = %v, want %v", testCase.preset, got, testCase.wantValid)
			}
			cfg := &config.GovernanceConfig{Preset: testCase.preset}
			cfg.ApplyPreset()
			if (cfg.Analyzers == nil) != (testCase.wantNilMap || !testCase.wantValid) {
				t.Errorf("Analyzers = %v after ApplyPreset(%q)", cfg.Analyzers, testCase.preset)
			}
		})
	}
}
