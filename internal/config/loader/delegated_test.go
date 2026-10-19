package loader

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

type delegatedCase struct {
	name         string
	private      string
	wantSeverity string
	wantOverride int
	wantErr      string
}

func delegatedCases() []delegatedCase {
	const named = ".git/pulsarules.yaml: "
	tune := func(rules string) string {
		return doc("overrides: [{files: [a], analyzers: {" + rules + "}}]")
	}
	return []delegatedCase{
		{
			name:         "severity in analyzers",
			private:      doc("analyzers: {dart-an: {severity: info}}"),
			wantSeverity: "info",
		},
		{name: "enabled in analyzers", private: doc("analyzers: {dart-an: {enabled: false}}")},
		{
			name:         "override keyed by the id",
			private:      tune("dart-an: {severity: error}"),
			wantOverride: 1,
		},
		{
			name:         "override keyed by a sub id",
			private:      tune("dart-an/rule: {enabled: false}"),
			wantOverride: 1,
		},
		{
			name:    "an id in neither file",
			private: doc("analyzers: {nope: {severity: info}}"),
			wantErr: named + "analyzers.nope: unknown analyzer",
		},
		{
			name:    "an override id in neither file",
			private: tune("nope: {enabled: false}"),
			wantErr: named + "overrides[0].analyzers.nope: unknown analyzer",
		},
	}
}

func TestPrivateTunesBaseDelegated(t *testing.T) {
	t.Parallel()

	base := doc("delegated: [{id: dart-an, run: [dart, analyze], format: dart-analyze-json}]")
	for _, testCase := range delegatedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: base, private: testCase.private, git: true}
			result := loadWithin(t, spec.build(t), "")
			if testCase.wantErr != "" {
				assertErrorHas(t, result.err, []string{testCase.wantErr})
				return
			}
			if result.err != nil {
				t.Fatalf("Load() error = %v", result.err)
			}
			cfg, err := Build(result.file, "")
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if got := cfg.Severities["dart-an"]; got != testCase.wantSeverity {
				t.Fatalf("severity = %q, want %q", got, testCase.wantSeverity)
			}
			if len(cfg.Overrides) != testCase.wantOverride {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverride)
			}
		})
	}
}

func TestPrivateRulesWithoutBase(t *testing.T) {
	t.Parallel()

	declare := func(ids ...string) *projectfile.File {
		file := &projectfile.File{}
		for _, id := range ids {
			file.Delegated = append(file.Delegated, projectfile.DelegatedEntry{ID: id})
		}
		return file
	}
	testCases := []struct {
		name      string
		base      *projectfile.File
		wantExtra []string
	}{
		{"no base", nil, nil},
		{"one delegated id", declare("x"), []string{"x"}},
		{"two delegated ids", declare("x", "y"), []string{"x", "y"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := privateRules(testCase.base).AnalyzerIDs
			want := slices.Concat(configRules().AnalyzerIDs, testCase.wantExtra)
			if !slices.Equal(got, want) {
				t.Fatalf("AnalyzerIDs = %d ids, want %d", len(got), len(want))
			}
		})
	}
}

func TestFailedBaseDoesNotBlameDelegatedIDs(t *testing.T) {
	t.Parallel()

	const declared = "delegated: [{id: dart-an, run: [dart, analyze], format: dart-analyze-json}]"
	testCases := []struct {
		name, private string
		wantAlso      bool
	}{
		{"private tunes the id", doc("analyzers: {dart-an: {severity: info}}"), false},
		{
			"private override keyed by it",
			doc("overrides: [{files: [a], analyzers: {dart-an/x: {enabled: false}}}]"), false,
		},
		{"another private problem is still reported", doc("bogus: 1"), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: doc("bogus: 1", declared), private: testCase.private, git: true}
			_, findings := forRoot(spec.build(t))
			assertConfigFinding(
				t, findings, projectfile.Name, ".pulsarules.yaml: line 2: bogus")
			named := strings.Contains(findings[0].Message, ".git/pulsarules.yaml")
			if named != testCase.wantAlso {
				t.Fatalf("message = %q, names the private file = %v, want %v",
					findings[0].Message, named, testCase.wantAlso)
			}
		})
	}
}
