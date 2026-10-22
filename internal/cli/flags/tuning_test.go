package flags

import (
	"flag"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestTuningValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		tuning  Tuning
		wantErr string
	}{
		{name: "zero value is valid"},
		{name: "known severity", tuning: Tuning{TypographicSeverity: "warning"}},
		{name: "extra types", tuning: Tuning{BranchExtraTypes: "spike,experiment"}},
		{
			name:    "severity typo names its flag",
			tuning:  Tuning{TypographicSeverity: "fatal"},
			wantErr: "invalid --typographic-severity: ",
		},
		{
			name:   "extra types uppercase names its flag and form",
			tuning: Tuning{BranchExtraTypes: "Release"},
			wantErr: `invalid --branch-extra-types: "Release" is not a comma-separated list ` +
				`of lowercase names, e.g. spike,experiment`,
		},
		{
			name:    "extra types spaces are refused",
			tuning:  Tuning{BranchExtraTypes: " a , b"},
			wantErr: `invalid --branch-extra-types: " a , b" is not`,
		},
		{
			name:    "extra types empty entry is refused",
			tuning:  Tuning{BranchExtraTypes: "a,,b"},
			wantErr: `invalid --branch-extra-types: "a,,b" is not`,
		},
		{
			name:    "severity is checked before extra types",
			tuning:  Tuning{TypographicSeverity: "fatal", BranchExtraTypes: "Bad"},
			wantErr: "invalid --typographic-severity: ",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := testCase.tuning.Validate()
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if err == nil || !strings.HasPrefix(err.Error(), testCase.wantErr) {
				t.Fatalf("Validate err = %v, want prefix %q", err, testCase.wantErr)
			}
		})
	}
}

func TestTuningApply(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		tuning       Tuning
		wantSeverity bool
		wantParam    any
	}{
		{name: "zero value changes nothing"},
		{
			name:         "severity becomes a rule",
			tuning:       Tuning{TypographicSeverity: "info"},
			wantSeverity: true,
		},
		{
			name:      "extra types become a param",
			tuning:    Tuning{BranchExtraTypes: "spike"},
			wantParam: "spike",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := &config.GovernanceConfig{}
			testCase.tuning.Apply(cfg)
			hasRule := len(cfg.SeverityRules) > 0 || len(cfg.Overrides) > 0
			if hasRule != testCase.wantSeverity {
				t.Errorf("severity rule present = %v, want %v", hasRule, testCase.wantSeverity)
			}
			if got := cfg.Analyzers["branch-name"].Params["extra_types"]; got != testCase.wantParam {
				t.Errorf("extra_types = %v, want %v", got, testCase.wantParam)
			}
		})
	}
}

func TestTuningBind(t *testing.T) {
	t.Parallel()

	for _, purpose := range []Purpose{Analyze, Bake} {
		var tuning Tuning
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		tuning.Bind(fs, purpose)
		args := []string{"--typographic-severity", "info", "--branch-extra-types", "a,b"}
		if err := fs.Parse(args); err != nil {
			t.Fatalf("purpose %d Parse: %v", purpose, err)
		}
		if tuning.TypographicSeverity != "info" || tuning.BranchExtraTypes != "a,b" {
			t.Errorf("purpose %d tuning = %+v", purpose, tuning)
		}
	}
}

func TestTuningBind_HelpTextFollowsThePurpose(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		purpose  Purpose
		severity string
		extra    string
	}{
		{
			name:     "analyze",
			purpose:  Analyze,
			severity: "how hard a typographic-marker finding lands (`error|warning|info`)",
			extra:    "comma-separated branch types (`a,b`) allowed beyond the Conventional Commit",
		},
		{
			name:     "bake",
			purpose:  Bake,
			severity: "bake into the git hooks how hard a typographic-marker finding lands",
			extra:    "bake into the pre-push hook extra branch types a project allows",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var tuning Tuning
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			tuning.Bind(fs, testCase.purpose)
			if got := fs.Lookup(
				"typographic-severity",
			).Usage; !strings.HasPrefix(
				got,
				testCase.severity,
			) {
				t.Errorf("severity usage = %q, want prefix %q", got, testCase.severity)
			}
			if got := fs.Lookup(
				"branch-extra-types",
			).Usage; !strings.HasPrefix(
				got,
				testCase.extra,
			) {
				t.Errorf("extra types usage = %q, want prefix %q", got, testCase.extra)
			}
		})
	}
}
