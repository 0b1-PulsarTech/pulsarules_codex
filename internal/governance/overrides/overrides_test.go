package overrides

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

func disable() pass.OverrideConfig {
	off := false
	return pass.OverrideConfig{Enabled: &off}
}

func enable() pass.OverrideConfig {
	on := true
	return pass.OverrideConfig{Enabled: &on}
}

type applyCase struct {
	name  string
	owner string
	rules []pass.Override
	want  []finding.Finding
}

func sample() []finding.Finding {
	return []finding.Finding{
		{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: finding.SeverityWarning},
		{AnalyzerID: "file-size", File: "web/lib/b.dart", Severity: finding.SeverityWarning},
		{AnalyzerID: "single-letter-name", File: "web/test/a.dart"},
		{
			AnalyzerID: "golangci-lint/errcheck",
			File:       "web/test/c.go",
			Severity:   finding.SeverityError,
		},
		{AnalyzerID: "branch-name", Severity: finding.SeverityError},
	}
}

func applyPerOwner(
	findings []finding.Finding,
	owner string,
	rules []pass.Override,
) []finding.Finding {
	got := make([]finding.Finding, 0, len(findings))
	for _, hit := range findings {
		base, _, _ := strings.Cut(hit.AnalyzerID, "/")
		got = append(got, Apply([]finding.Finding{hit}, cmp.Or(owner, base), rules)...)
	}
	return got
}

func runApplyCases(t *testing.T, testCases []applyCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := applyPerOwner(sample(), testCase.owner, testCase.rules)
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("Apply() = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestApplyDropAndRewrite(t *testing.T) {
	t.Parallel()

	input := sample()
	testGlob := []string{"web/test/**"}
	runApplyCases(t, []applyCase{
		{"no rules", "", nil, input},
		{
			"drop in matching files only",
			"",
			[]pass.Override{{
				Files:     glob.CompileSet(testGlob),
				Analyzers: map[string]pass.OverrideConfig{"single-letter-name": disable()},
			}},
			[]finding.Finding{input[0], input[1], input[3], input[4]},
		},
		{
			"rewrite in matching files only",
			"",
			[]pass.Override{{
				Files:     glob.CompileSet(testGlob),
				Analyzers: map[string]pass.OverrideConfig{"file-size": {Severity: "info"}},
			}},
			[]finding.Finding{
				{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: finding.SeverityInfo},
				input[1], input[2], input[3], input[4],
			},
		},
		{
			"later entry wins",
			"",
			[]pass.Override{
				{Files: glob.CompileSet(testGlob), Analyzers: map[string]pass.OverrideConfig{
					"single-letter-name": disable(),
					"file-size":          {Severity: "error"},
				}},
				{
					Files: glob.CompileSet([]string{"web/test/a.dart"}),
					Analyzers: map[string]pass.OverrideConfig{
						"single-letter-name": enable(),
						"file-size":          {Severity: "info"},
					},
				},
			},
			[]finding.Finding{
				{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: finding.SeverityInfo},
				input[1], input[2], input[3], input[4],
			},
		},
	})
}

func TestApplyScopeAndIDs(t *testing.T) {
	t.Parallel()

	input := sample()
	runApplyCases(t, slices.Concat(scopeCases(input), idCases(input)))
}

func scopeCases(input []finding.Finding) []applyCase {
	return []applyCase{
		{
			"no files scope reaches fileless findings",
			"",
			[]pass.Override{{
				Analyzers: map[string]pass.OverrideConfig{"branch-name": {Severity: "warning"}},
			}},
			[]finding.Finding{
				input[0], input[1], input[2], input[3],
				{AnalyzerID: "branch-name", Severity: finding.SeverityWarning},
			},
		},
		{
			"globs never match a fileless finding",
			"",
			[]pass.Override{{
				Files:     glob.CompileSet([]string{"**"}),
				Analyzers: map[string]pass.OverrideConfig{"branch-name": disable()},
			}},
			input,
		},
		{
			"an uncompilable glob matches nothing",
			"",
			[]pass.Override{{
				Files:     glob.CompileSet([]string{"a//b"}),
				Analyzers: map[string]pass.OverrideConfig{"file-size": disable()},
			}},
			input,
		},
	}
}

func idCases(input []finding.Finding) []applyCase {
	testGlob := []string{"web/test/**"}
	return []applyCase{
		{
			"base id matches a sub id and the exact sub id matches too",
			"",
			[]pass.Override{
				{Files: glob.CompileSet(testGlob), Analyzers: map[string]pass.OverrideConfig{
					"golangci-lint": {Severity: "warning"},
				}},
				{Files: glob.CompileSet(testGlob), Analyzers: map[string]pass.OverrideConfig{
					"golangci-lint/errcheck": {Severity: "info"},
				}},
			},
			[]finding.Finding{
				input[0], input[1], input[2],
				{AnalyzerID: "golangci-lint/errcheck", File: "web/test/c.go"},
				input[4],
			},
		},
		{
			"the prefix before a slash matches when the owner is another analyzer",
			"naming",
			[]pass.Override{{
				Files:     glob.CompileSet(testGlob),
				Analyzers: map[string]pass.OverrideConfig{"golangci-lint": {Severity: "warning"}},
			}},
			[]finding.Finding{
				input[0], input[1], input[2],
				{
					AnalyzerID: "golangci-lint/errcheck",
					File:       "web/test/c.go",
					Severity:   finding.SeverityWarning,
				},
				input[4],
			},
		},
	}
}

func TestApplyKeepsInputUntouched(t *testing.T) {
	t.Parallel()

	input := []finding.Finding{
		{AnalyzerID: "file-size", File: "a.go", Severity: finding.SeverityError},
	}
	rules := []pass.Override{{
		Files:     glob.CompileSet([]string{"**"}),
		Analyzers: map[string]pass.OverrideConfig{"file-size": {Severity: "info"}},
	}}
	Apply(input, "", rules)
	if input[0].Severity != finding.SeverityError {
		t.Fatalf("input severity = %v, want it left alone", input[0].Severity)
	}
}

func TestApplyMatchesTheOwningAnalyzer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		key   string
		id    string
		owner string
		want  finding.Severity
	}{
		{
			"owner key rewrites the finer finding id",
			"commit-lint", "commit-emoji", "commit-lint", finding.SeverityInfo,
		},
		{
			"another owner leaves it untouched",
			"commit-lint", "commit-emoji", "naming", finding.SeverityError,
		},
		{
			"a bare key never captures another analyzer's id",
			"ai-frontmatter", "ai-frontmatter", "text-markers", finding.SeverityError,
		},
		{
			"a bare key matches its own analyzer",
			"text-markers", "text-markers", "text-markers", finding.SeverityInfo,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rules := []pass.Override{{
				Analyzers: map[string]pass.OverrideConfig{testCase.key: {Severity: "info"}},
			}}
			input := []finding.Finding{{AnalyzerID: testCase.id, Severity: finding.SeverityError}}
			got := Apply(input, testCase.owner, rules)
			if len(got) != 1 || got[0].Severity != testCase.want {
				t.Fatalf("Apply() = %+v, want severity %v", got, testCase.want)
			}
		})
	}
}
