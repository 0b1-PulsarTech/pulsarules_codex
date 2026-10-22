package projectfile

import (
	"reflect"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func fullProjectFile(enabled, disabled *bool) *File {
	return &File{
		IncludeGenerated: enabled,
		SkipDirs:         []string{"gen"},
		Tests:            []string{"t/**"},
		Analyzers: map[string]AnalyzerEntry{
			"alpha": {Enabled: disabled, Severity: "info", Params: map[string]any{"key": 1}},
		},
		Overrides: []OverrideEntry{{
			Files: []string{"web/**"},
			Analyzers: map[string]AnalyzerEntry{
				"beta": {Enabled: disabled, Severity: "warning", Params: map[string]any{"cap": 5}},
			},
		}},
		Delegated: []DelegatedEntry{{
			ID: "gamma", Dir: "sub", Format: "json", Rule: "lint", Run: []string{"go", "vet"},
		}},
	}
}

func TestApplyToConvertsEveryField(t *testing.T) {
	t.Parallel()

	enabled, disabled := true, false
	file := fullProjectFile(&enabled, &disabled)
	cfg := config.Defaults()
	if err := file.ApplyTo(cfg); err != nil {
		t.Fatalf("ApplyTo() error = %v", err)
	}
	file.Delegated[0].Run[0] = "mutated"
	file.Tests[0] = "mutated"
	file.Overrides[0].Analyzers["beta"].Params["cap"] = 99

	if !cfg.IncludeGenerated {
		t.Error("IncludeGenerated = false, want true")
	}
	if !reflect.DeepEqual(cfg.SkipDirs, []string{"gen"}) {
		t.Errorf("SkipDirs = %v", cfg.SkipDirs)
	}
	if !reflect.DeepEqual(cfg.TestGlobs, []string{"t/**"}) {
		t.Errorf("TestGlobs = %v", cfg.TestGlobs)
	}
	wantAnalyzer := pass.AnalyzerConfig{Enabled: false, Params: map[string]any{"key": 1}}
	if got := cfg.Analyzers["alpha"]; !reflect.DeepEqual(got, wantAnalyzer) {
		t.Errorf("alpha = %+v, want %+v", got, wantAnalyzer)
	}
	wantCommand := pass.DelegatedCommand{
		ID: "gamma", Dir: "sub", Format: "json", Rule: "lint", Run: []string{"go", "vet"},
	}
	if !reflect.DeepEqual(cfg.Delegated, []pass.DelegatedCommand{wantCommand}) {
		t.Errorf("Delegated = %+v, want %+v", cfg.Delegated, wantCommand)
	}
	checkConvertedOverrides(t, cfg)
}

func checkConvertedOverrides(t *testing.T, cfg *config.GovernanceConfig) {
	t.Helper()
	if len(cfg.SeverityRules) != 1 || cfg.SeverityRules[0].Analyzers["alpha"].Severity != "info" {
		t.Errorf("SeverityRules = %+v, want alpha info", cfg.SeverityRules)
	}
	if len(cfg.Overrides) != 1 {
		t.Fatalf("Overrides = %+v, want one", cfg.Overrides)
	}
	disabled := false
	want := pass.OverrideConfig{
		Enabled:  &disabled,
		Severity: "warning",
		Params:   map[string]any{"cap": 5},
	}
	if got := cfg.Overrides[0].Analyzers["beta"]; !reflect.DeepEqual(got, want) {
		t.Errorf("beta override = %+v, want %+v", got, want)
	}
	if !cfg.Overrides[0].Covers("web/a.go") || cfg.Overrides[0].Covers("api/a.go") {
		t.Errorf("override files = %+v, want web/** only", cfg.Overrides[0].Files)
	}
}
