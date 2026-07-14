package config

import (
	"testing"
)

func TestApplyPreset_Strict(t *testing.T) {
	t.Parallel()

	cfg := Defaults()
	cfg.Preset = PresetStrict
	cfg.ApplyPreset()

	if !isEnabled(cfg, "file-size") {
		t.Error("file-size should be enabled in strict preset")
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

func isEnabled(cfg *GovernanceConfig, analyzerID string) bool {
	ac, ok := cfg.Analyzers[analyzerID]
	if !ok {
		return true
	}
	return ac.Enabled
}
