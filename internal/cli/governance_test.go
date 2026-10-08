package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

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
			got := cfg.Analyzers["golangci-lint"].Params["config_path"]
			if got != testCase.wantPath {
				t.Errorf("config_path = %v, want %v", got, testCase.wantPath)
			}
		})
	}
}

func TestGovernanceConfig_InvalidPreset(t *testing.T) {
	t.Parallel()

	_, err := governanceConfig(&cliopts.Options{Preset: "stict"})
	if err == nil {
		t.Fatal("governanceConfig: expected an error for an unknown preset")
	}
	for _, want := range []string{
		config.PresetRecommended,
		config.PresetStrict,
		config.PresetMinimal,
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name valid preset %q", err, want)
		}
	}
}

func TestValidateScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		scope   string
		wantErr bool
	}{
		{name: "empty defers to the flag default", scope: ""},
		{name: "full is accepted", scope: "full"},
		{name: "commit is accepted", scope: "commit"},
		{name: "changed is accepted", scope: "changed"},
		{name: "typo is rejected", scope: "fulll", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateScope(testCase.scope)
			if testCase.wantErr {
				if err == nil {
					t.Fatalf("validateScope(%q): expected an error", testCase.scope)
				}
				if !strings.Contains(err.Error(), "full") ||
					!strings.Contains(err.Error(), "commit") {
					t.Errorf("error %q does not name the valid scopes", err)
				}
				return
			}
			if err != nil {
				t.Errorf("validateScope(%q): unexpected error: %v", testCase.scope, err)
			}
		})
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
				t.Errorf("suppressedClause(%d) = %q, want %q",
					testCase.suppressed, got, testCase.want)
			}
		})
	}
}

func TestGovernanceConfig_StrictLowersFileSize(t *testing.T) {
	t.Parallel()

	cfg, err := governanceConfig(&cliopts.Options{Preset: config.PresetStrict})
	if err != nil {
		t.Fatalf("governanceConfig: %v", err)
	}
	if got := cfg.Analyzers["file-size"].Params["max_lines"]; got != 180 {
		t.Errorf("strict max_lines = %v, want 180", got)
	}
	if got := cfg.Analyzers["complexity"].Params["max_complexity"]; got != 10 {
		t.Errorf("strict max_complexity = %v, want 10", got)
	}
}

func TestGovernanceConfig_TypographicSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		wantErr  bool
		want     any
	}{
		{name: "unset leaves the params alone", want: nil},
		{name: "warning reaches the analyzer", severity: "warning", want: "warning"},
		{name: "info reaches the analyzer", severity: "info", want: "info"},
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
			params := cfg.Analyzers["typographic-markers"].Params
			if got := params["severity"]; got != testCase.want {
				t.Errorf("severity param = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestGovernanceConfig_BranchExtraTypes(t *testing.T) {
	t.Parallel()

	const wantMessage = `invalid --branch-extra-types: "Release" is not a comma-separated list ` +
		`of lowercase names, e.g. spike,experiment`

	testCases := []struct {
		name    string
		spec    string
		wantErr string
		wantSet any
	}{
		{name: "unset leaves the params alone", wantSet: nil},
		{
			name: "a list reaches the analyzer", spec: "spike,experiment",
			wantSet: "spike,experiment",
		},
		{name: "uppercase names the flag and the form", spec: "Release", wantErr: wantMessage},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := governanceConfig(&cliopts.Options{BranchExtraTypes: testCase.spec})
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("err = %v, want one naming %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("governanceConfig: %v", err)
			}
			if got := cfg.Analyzers["branch-name"].Params["extra_types"]; got != testCase.wantSet {
				t.Errorf("extra_types param = %v, want %v", got, testCase.wantSet)
			}
		})
	}
}
