package branchname

import (
	"errors"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// repoStub reports a fixed branch. One method, so it is hand-rolled.
type repoStub struct {
	branch string
	err    error
}

func (r repoStub) CurrentBranch() (string, error) { return r.branch, r.err }

var errStub = errors.New("git unavailable")

var _ branchReader = repoStub{}

// TestAnalyze covers what the branch check accepts, what it blocks, and the
// states where it must say nothing at all.
func TestAnalyze(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		branch      string
		repoErr     error
		extraTypes  string
		wantFinding bool
	}{
		{name: "a conventional type passes", branch: "feat/branch-name-check"},
		// This repo's own convention: every merge in its history is feature/<name>,
		// so the rule meant to protect it must not block it.
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
			extraTypes: "spike,poc",
		},
		{
			name:        "an extra type does not whitelist everything",
			branch:      "claude/x",
			extraTypes:  "release",
			wantFinding: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &core.AnalysisContext{}
			if testCase.extraTypes != "" {
				ctx.Config = &core.AnalysisConfig{Analyzers: map[string]core.AnalyzerConfig{
					analyzerID: {
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
				if got[0].Severity != core.SeverityError {
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

// TestAnalyze_NilRepo asserts a pipeline built without git says nothing rather
// than panicking.
func TestAnalyze_NilRepo(t *testing.T) {
	t.Parallel()

	if got := NewAnalyzer(nil).Analyze(&core.AnalysisContext{}); len(got) != 0 {
		t.Errorf("findings = %+v, want none", got)
	}
}

// TestValidExtraTypes guards the boundary where the value is written into a
// generated shell script: anything outside the allowed alphabet is refused, so
// nothing has to be quoted and hoped for at the point the hook runs.
func TestValidExtraTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		spec string
		want bool
	}{
		{name: "empty is allowed", spec: "", want: true},
		{name: "one name", spec: "release", want: true},
		{name: "several names", spec: "release,hotfix", want: true},
		{name: "digits and dashes", spec: "release-2,hot-fix9", want: true},
		{name: "a trailing comma leaves an empty entry", spec: "release,"},
		{name: "a leading comma leaves an empty entry", spec: ",release"},
		{name: "spaces are refused rather than trimmed", spec: "release, hotfix"},
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

			if got := ValidExtraTypes(testCase.spec); got != testCase.want {
				t.Errorf("ValidExtraTypes(%q) = %v, want %v", testCase.spec, got, testCase.want)
			}
		})
	}
}
