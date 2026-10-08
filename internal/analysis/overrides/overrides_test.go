package overrides

import (
	"cmp"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func disable() core.OverrideConfig {
	off := false
	return core.OverrideConfig{Enabled: &off}
}

func enable() core.OverrideConfig {
	on := true
	return core.OverrideConfig{Enabled: &on}
}

type applyCase struct {
	name  string
	owner string
	rules []core.Override
	want  []core.Finding
}

func sample() []core.Finding {
	return []core.Finding{
		{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: core.SeverityWarning},
		{AnalyzerID: "file-size", File: "web/lib/b.dart", Severity: core.SeverityWarning},
		{AnalyzerID: "single-letter-name", File: "web/test/a.dart"},
		{
			AnalyzerID: "golangci-lint/errcheck",
			File:       "web/test/c.go",
			Severity:   core.SeverityError,
		},
		{AnalyzerID: "branch-name", Severity: core.SeverityError},
	}
}

func applyPerOwner(findings []core.Finding, owner string, rules []core.Override) []core.Finding {
	got := make([]core.Finding, 0, len(findings))
	for _, finding := range findings {
		base, _, _ := strings.Cut(finding.AnalyzerID, "/")
		got = append(got, Apply([]core.Finding{finding}, cmp.Or(owner, base), rules)...)
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
			[]core.Override{{
				Files:     testGlob,
				Analyzers: map[string]core.OverrideConfig{"single-letter-name": disable()},
			}},
			[]core.Finding{input[0], input[1], input[3], input[4]},
		},
		{
			"rewrite in matching files only",
			"",
			[]core.Override{{
				Files:     testGlob,
				Analyzers: map[string]core.OverrideConfig{"file-size": {Severity: "info"}},
			}},
			[]core.Finding{
				{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: core.SeverityInfo},
				input[1], input[2], input[3], input[4],
			},
		},
		{
			"later entry wins",
			"",
			[]core.Override{
				{Files: testGlob, Analyzers: map[string]core.OverrideConfig{
					"single-letter-name": disable(),
					"file-size":          {Severity: "error"},
				}},
				{Files: []string{"web/test/a.dart"}, Analyzers: map[string]core.OverrideConfig{
					"single-letter-name": enable(),
					"file-size":          {Severity: "info"},
				}},
			},
			[]core.Finding{
				{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: core.SeverityInfo},
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

func scopeCases(input []core.Finding) []applyCase {
	return []applyCase{
		{
			"no files scope reaches fileless findings",
			"",
			[]core.Override{{
				Analyzers: map[string]core.OverrideConfig{"branch-name": {Severity: "warning"}},
			}},
			[]core.Finding{
				input[0], input[1], input[2], input[3],
				{AnalyzerID: "branch-name", Severity: core.SeverityWarning},
			},
		},
		{
			"globs never match a fileless finding",
			"",
			[]core.Override{{
				Files:     []string{"**"},
				Analyzers: map[string]core.OverrideConfig{"branch-name": disable()},
			}},
			input,
		},
		{
			"an uncompilable glob matches nothing",
			"",
			[]core.Override{{
				Files:     []string{"a//b"},
				Analyzers: map[string]core.OverrideConfig{"file-size": disable()},
			}},
			input,
		},
	}
}

func idCases(input []core.Finding) []applyCase {
	testGlob := []string{"web/test/**"}
	return []applyCase{
		{
			"base id matches a sub id and the exact sub id matches too",
			"",
			[]core.Override{
				{Files: testGlob, Analyzers: map[string]core.OverrideConfig{
					"golangci-lint": {Severity: "warning"},
				}},
				{Files: testGlob, Analyzers: map[string]core.OverrideConfig{
					"golangci-lint/errcheck": {Severity: "info"},
				}},
			},
			[]core.Finding{
				input[0], input[1], input[2],
				{AnalyzerID: "golangci-lint/errcheck", File: "web/test/c.go"},
				input[4],
			},
		},
		{
			"the prefix before a slash matches when the owner is another analyzer",
			"naming",
			[]core.Override{{
				Files:     testGlob,
				Analyzers: map[string]core.OverrideConfig{"golangci-lint": {Severity: "warning"}},
			}},
			[]core.Finding{
				input[0], input[1], input[2],
				{
					AnalyzerID: "golangci-lint/errcheck",
					File:       "web/test/c.go",
					Severity:   core.SeverityWarning,
				},
				input[4],
			},
		},
	}
}

func TestApplyKeepsInputUntouched(t *testing.T) {
	t.Parallel()

	input := []core.Finding{{AnalyzerID: "file-size", File: "a.go", Severity: core.SeverityError}}
	rules := []core.Override{{
		Files:     []string{"**"},
		Analyzers: map[string]core.OverrideConfig{"file-size": {Severity: "info"}},
	}}
	Apply(input, "", rules)
	if input[0].Severity != core.SeverityError {
		t.Fatalf("input severity = %v, want it left alone", input[0].Severity)
	}
}

func TestParseSeverity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want core.Severity
		ok   bool
	}{
		{"error", core.SeverityError, true},
		{"warning", core.SeverityWarning, true},
		{"info", core.SeverityInfo, true},
		{"", core.SeverityInfo, false},
		{"loud", core.SeverityInfo, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseSeverity(testCase.name)
			if got != testCase.want || ok != testCase.ok {
				t.Fatalf("parseSeverity(%q) = %v, %v", testCase.name, got, ok)
			}
		})
	}
}

func TestApplyMatchesTheOwningAnalyzer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		key   string
		id    string
		owner string
		want  core.Severity
	}{
		{
			"owner key rewrites the finer finding id",
			"commit-lint", "commit-emoji", "commit-lint", core.SeverityInfo,
		},
		{
			"another owner leaves it untouched",
			"commit-lint", "commit-emoji", "naming", core.SeverityError,
		},
		{
			"a bare key never captures another analyzer's id",
			"ai-frontmatter", "ai-frontmatter", "text-markers", core.SeverityError,
		},
		{
			"a bare key matches its own analyzer",
			"text-markers", "text-markers", "text-markers", core.SeverityInfo,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rules := []core.Override{{
				Analyzers: map[string]core.OverrideConfig{testCase.key: {Severity: "info"}},
			}}
			input := []core.Finding{{AnalyzerID: testCase.id, Severity: core.SeverityError}}
			got := Apply(input, testCase.owner, rules)
			if len(got) != 1 || got[0].Severity != testCase.want {
				t.Fatalf("Apply() = %+v, want severity %v", got, testCase.want)
			}
		})
	}
}
