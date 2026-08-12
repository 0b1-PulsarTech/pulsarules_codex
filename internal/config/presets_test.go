package config

import (
	"testing"
)

func TestApplyPreset_RecommendedDefaults(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetRecommended
	cfg.ApplyPreset()

	if isEnabled(cfg, "complexity") != true {
		t.Error("complexity should be enabled in recommended")
	}
}

func TestApplyPreset_Minimal(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetMinimal
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
}

func TestApplyPreset_Strict(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetStrict
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

	cfg := Defaults()
	cfg.Preset = "unknown"
	cfg.ApplyPreset()

	if !isEnabled(cfg, "complexity") {
		t.Error("complexity should still be enabled after unknown preset")
	}
}

func TestApplyPreset_StrictOverridesMaxLinesParam(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetStrict
	cfg.ApplyPreset()

	maxLines := cfg.Analyzers["file-size"].Params["max_lines"]
	if maxLines != 180 {
		t.Errorf("expected max_lines=180 in strict preset, got %v", maxLines)
	}
}

func TestValidPreset(t *testing.T) {
	t.Parallel()

	if !ValidPreset(PresetRecommended) {
		t.Error("recommended should be valid")
	}
	if !ValidPreset(PresetStrict) {
		t.Error("strict should be valid")
	}
	if !ValidPreset(PresetMinimal) {
		t.Error("minimal should be valid")
	}
	if ValidPreset("unknown") {
		t.Error("unknown should not be valid")
	}
}

func TestPresets(t *testing.T) {
	t.Parallel()

	const wantPresetCount = 3

	list := Presets()
	if len(list) != wantPresetCount {
		t.Fatalf("expected %d presets, got %d: %v", wantPresetCount, len(list), list)
	}
	if list[0] != PresetRecommended {
		t.Errorf("first preset should be recommended, got %q", list[0])
	}
}
