package branch

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type repoStub struct {
	branch string
	err    error
}

func (repo repoStub) CurrentBranch() (string, error) { return repo.branch, repo.err }

var errStub = errors.New("git unavailable")

var _ branchReader = repoStub{}

func TestAnalyze(t *testing.T) {
	t.Parallel()

	testCases := analyzeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &pass.Context{}
			if testCase.extraTypes != "" {
				ctx.Config = &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
					ID: {
						Enabled: true,
						Params:  map[string]any{"extra_types": testCase.extraTypes},
					},
				}}
			}
			got := NewAnalyzer(
				repoStub{branch: testCase.branch, err: testCase.repoErr},
			).Analyze(ctx)
			if testCase.wantFinding {
				if len(got) != 1 {
					t.Fatalf("findings = %+v, want exactly one", got)
				}
				if got[0].Severity != finding.SeverityError {
					t.Errorf("severity = %v, want error (it must block the push)", got[0].Severity)
				}
				if !strings.Contains(got[0].Message, testCase.branch) {
					t.Errorf("message %q does not name the branch", got[0].Message)
				}
				return
			}
			if len(got) != 0 {
				t.Errorf("findings = %+v, want none", got)
			}
		})
	}
}

type analyzeCase struct {
	name        string
	branch      string
	repoErr     error
	extraTypes  string
	wantFinding bool
}

func analyzeCases() []analyzeCase {
	return []analyzeCase{
		{name: "a conventional type passes", branch: "feat/branch-name-check"},
		{name: "a gitflow feature line passes", branch: "feature/add_branch-check"},
		{name: "a gitflow release line passes", branch: "release/1.2"},
		{name: "a gitflow hotfix line passes", branch: "hotfix/urgent"},
		{name: "a scoped type passes", branch: "fix(hook)/worktree-hooks-dir"},
		{name: "every allowed type passes", branch: "revert/bad-merge"},
		{name: "main is exempt", branch: "main"},
		{name: "master is exempt", branch: "master"},
		{name: "develop is exempt", branch: "develop"},
		{name: "a detached head names no branch", branch: ""},
		{name: "an unreadable repo stays silent", branch: "claude/x", repoErr: errStub},
		{
			name:        "a tool-generated prefix is blocked",
			branch:      "claude/code-improvements-planning",
			wantFinding: true,
		},
		{name: "no slash at all is blocked", branch: "featbranch", wantFinding: true},
		{name: "a type with an empty description is blocked", branch: "feat/", wantFinding: true},
		{name: "an unknown type is blocked", branch: "spike/idea", wantFinding: true},
		{name: "an unclosed scope is blocked", branch: "fix(hook/thing", wantFinding: true},
		{
			name:       "an unknown type passes once the project adds it",
			branch:     "spike/idea",
			extraTypes: "spike,experiment",
		},
		{
			name:        "an extra type does not whitelist everything",
			branch:      "claude/x",
			extraTypes:  "release",
			wantFinding: true,
		},
	}
}

func TestAnalyze_NilRepo(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer(nil).Analyze(&pass.Context{}); len(got) != 0 {
		t.Errorf("findings = %+v, want none", got)
	}
}

func TestAnalyze_SeverityParam(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		severity string
		want     finding.Severity
	}{
		{name: "unset keeps the blocking default", want: finding.SeverityError},
		{
			name:     "configured warning overrides the default",
			severity: "warning",
			want:     finding.SeverityWarning,
		},
		{
			name:     "unrecognized value keeps the blocking default",
			severity: "bogus",
			want:     finding.SeverityError,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &pass.Context{}
			if testCase.severity != "" {
				ctx.Config = &pass.AnalysisConfig{Analyzers: map[string]pass.AnalyzerConfig{
					ID: {
						Enabled: true,
						Params:  map[string]any{"severity": testCase.severity},
					},
				}}
			}
			got := NewAnalyzer(repoStub{branch: "spike/idea"}).Analyze(ctx)
			if len(got) != 1 {
				t.Fatalf("findings = %+v, want exactly one", got)
			}
			if got[0].Severity != testCase.want {
				t.Errorf("severity = %v, want %v", got[0].Severity, testCase.want)
			}
		})
	}
}

func TestValidExtraTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		spec string
		want bool
	}{
		{name: "empty is allowed", spec: "", want: true},
		{name: "one name", spec: "release", want: true},
		{name: "several names", spec: "spike,experiment", want: true},
		{name: "digits and dashes", spec: "release-2,hot-fix9", want: true},
		{name: "a trailing comma leaves an empty entry", spec: "release,"},
		{name: "a leading comma leaves an empty entry", spec: ",release"},
		{name: "spaces are refused rather than trimmed", spec: "spike, experiment"},
		{name: "uppercase is refused", spec: "Release"},
		{name: "a command substitution is refused", spec: "release;rm -rf /"},
		{name: "a backtick is refused", spec: "a`id`"},
		{name: "a quote is refused", spec: `a"b`},
		{name: "a dollar sign is refused", spec: "a$HOME"},
		{name: "a newline is refused", spec: "release\nhotfix"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := validExtraTypes(testCase.spec); got != testCase.want {
				t.Errorf("validExtraTypes(%q) = %v, want %v", testCase.spec, got, testCase.want)
			}
		})
	}
}

func TestValidateExtraTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{name: "empty is allowed", spec: ""},
		{name: "a list is allowed", spec: "spike,experiment"},
		{name: "uppercase is refused", spec: "Release", wantErr: true},
		{name: "an empty entry is refused", spec: "release,", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateExtraTypes(testCase.spec)
			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("ValidateExtraTypes(%q) = %v, want nil", testCase.spec, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateExtraTypes(%q) = nil, want an error", testCase.spec)
			}
			if strings.Contains(err.Error(), "--") {
				t.Errorf("error %q names a CLI flag; the CLI callers add it", err)
			}
			wantQuoted := fmt.Sprintf("%q", testCase.spec)
			if !strings.Contains(err.Error(), wantQuoted) {
				t.Errorf("error %q does not quote the spec %s", err, wantQuoted)
			}
			if !strings.Contains(err.Error(), "comma-separated") {
				t.Errorf("error %q does not name the expected form", err)
			}
		})
	}
}
