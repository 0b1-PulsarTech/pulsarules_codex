package config

import (
	"testing"
)

func TestApplyPreset_Recommended(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetRecommended
	cfg.ApplyPreset()

	// recommended doesn't define overrides, so nothing changes
	if cfg.IsEnabled("complexity") != true {
		t.Error("complexity should be enabled in recommended")
	}
}

func TestApplyPreset_Minimal(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetMinimal
	cfg.ApplyPreset()

	if cfg.IsEnabled("complexity") {
		t.Error("complexity should be disabled in minimal preset")
	}
	if cfg.IsEnabled("arch-boundary") {
		t.Error("arch-boundary should be disabled in minimal preset")
	}
	if !cfg.IsEnabled("file-size") {
		t.Error("file-size should be enabled in minimal preset")
	}
	if !cfg.IsEnabled("naming") {
		t.Error("naming should be enabled in minimal preset")
	}
}

func TestApplyPreset_Strict(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetStrict
	cfg.ApplyPreset()

	if !cfg.IsEnabled("file-size") {
		t.Error("file-size should be enabled in strict preset")
	}
	if !cfg.IsEnabled("complexity") {
		t.Error("complexity should be enabled in strict preset")
	}
}

func TestApplyPreset_Unknown(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = "unknown"
	cfg.ApplyPreset() // should be a no-op

	if !cfg.IsEnabled("complexity") {
		t.Error("complexity should still be enabled after unknown preset")
	}
}

func TestApplyPreset_StrictParam(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetStrict
	cfg.ApplyPreset()

	v := cfg.Param("file-size", "max_lines", 999)
	if v != 180 {
		t.Errorf("expected max_lines=180 in strict preset, got %v", v)
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

	list := Presets()
	if len(list) != 3 {
		t.Fatalf("expected 3 presets, got %d: %v", len(list), list)
	}
	if list[0] != PresetRecommended {
		t.Errorf("first preset should be recommended, got %q", list[0])
	}
}
