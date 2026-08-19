package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

// TestRunGovernance_RequiresProjectDir pins the host-neutral env var check:
// generic Go reads PULSARULES_PROJECT_DIR, never a host's own variable, so a
// missing project dir must return errProjectDirRequired. It sets an
// environment variable, so it cannot run in parallel.
func TestRunGovernance_RequiresProjectDir(t *testing.T) {
	t.Setenv("PULSARULES_PROJECT_DIR", "")

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	err := runGovernance(inj, &cliopts.Options{})
	if !errors.Is(err, errProjectDirRequired) {
		t.Errorf("err = %v, want %v", err, errProjectDirRequired)
	}
}

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

			cfg, err := governanceConfig(testCase.opts)
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
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

// TestGovernanceConfig_InvalidPreset pins the fix for --preset accepting any
// string and silently falling back to recommended: an unrecognized name must
// fail, naming the valid set, instead of ApplyPreset's silent no-op letting
// the run continue on the default preset.
func TestGovernanceConfig_InvalidPreset(t *testing.T) {
	t.Parallel()

	_, err := governanceConfig(&cliopts.Options{Preset: "stict"})
	if err == nil {
		t.Fatal("governanceConfig: expected an error for an unknown preset")
	}
	for _, want := range []string{config.PresetRecommended, config.PresetStrict, config.PresetMinimal} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name valid preset %q", err, want)
		}
	}
}

func TestGovernanceConfig_IncludeGenerated(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts *cliopts.Options
		want bool
	}{
		{name: "suppressed by default", opts: &cliopts.Options{}, want: false},
		{
			name: "flag turns the filter off",
			opts: &cliopts.Options{IncludeGenerated: true},
			want: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := governanceConfig(testCase.opts)
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := cfg.IncludeGenerated; got != testCase.want {
				t.Errorf("IncludeGenerated = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSuppressedClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		suppressed int
		want       string
	}{
		{name: "nothing suppressed says nothing", suppressed: 0, want: ""},
		{
			name:       "one suppressed finding is still announced",
			suppressed: 1,
			want:       ", 1 suppressed in generated files",
		},
		{
			name:       "the count is stated in full",
			suppressed: 353,
			want:       ", 353 suppressed in generated files",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := suppressedClause(testCase.suppressed); got != testCase.want {
				t.Errorf(
					"suppressedClause(%d) = %q, want %q",
					testCase.suppressed,
					got,
					testCase.want,
				)
			}
		})
	}
}

// TestGovernanceConfig_StrictLowersFileSize pins the one preset override that
// still differs from the analyzer's own default, so deleting it as
// "redundant" fails here instead of silently loosening --preset strict.
func TestGovernanceConfig_StrictLowersFileSize(t *testing.T) {
	t.Parallel()

	cfg, err := governanceConfig(&cliopts.Options{Preset: config.PresetStrict})
	if err != nil {
		t.Fatalf("governanceConfig: %v", err)
	}
	if got := cfg.Param("file-size", "max_lines", 999); got != 180 {
		t.Errorf("strict max_lines = %v, want 180", got)
	}
	if got := cfg.Param("complexity", "max_complexity", 999); got != 10 {
		t.Errorf("strict max_complexity = %v, want 10", got)
	}
}

// TestGovernanceConfig_TypographicSeverity asserts the flag reaches the
// analyzer's params and that an unrecognized value is rejected by name rather
// than falling back silently to the blocking default.
func TestGovernanceConfig_TypographicSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		wantErr  bool
		wantSet  any
	}{
		{name: "unset leaves the params alone", wantSet: nil},
		{name: "warning reaches the analyzer", severity: "warning", wantSet: "warning"},
		{name: "info reaches the analyzer", severity: "info", wantSet: "info"},
		{name: "a typo is rejected", severity: "fatal", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := governanceConfig(&cliopts.Options{TypographicSeverity: testCase.severity})
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("governanceConfig(%q) err = nil, want an error", testCase.severity)
				}
				return
			}
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := cfg.Param("typographic-markers", "severity", nil); got != testCase.wantSet {
				t.Errorf("severity param = %v, want %v", got, testCase.wantSet)
			}
		})
	}
}
