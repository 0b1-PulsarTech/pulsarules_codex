package governance

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/packagevar"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/layout/importgroup"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/naming/identifier"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/signature/variadicindex"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type stubAnalyzer struct {
	findings []finding.Finding
}

func (stub stubAnalyzer) Analyze(_ *pass.Context) []finding.Finding {
	return stub.findings
}

func TestPipelineRun(t *testing.T) {
	t.Parallel()

	pipe := newPipeline(nil, []registry.Built{
		{ID: "a1", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "a1", Message: "issue 1", Severity: finding.SeverityWarning},
			},
		}},
		{ID: "a2", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "a2", Message: "issue 2", Severity: finding.SeverityError},
			},
		}},
	}, nil)

	ctx := &pass.Context{}
	findings := pipe.run(ctx, nil)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].AnalyzerID != "a1" {
		t.Errorf("first finding should be from a1, got %s", findings[0].AnalyzerID)
	}
	if findings[1].AnalyzerID != "a2" {
		t.Errorf("second finding should be from a2, got %s", findings[1].AnalyzerID)
	}
}

func TestPipelineDisabledAnalyzer(t *testing.T) {
	t.Parallel()

	cfg := &pass.AnalysisConfig{
		Analyzers: map[string]pass.AnalyzerConfig{
			"disabled": {Enabled: false},
		},
	}
	pipe := newPipeline(cfg, []registry.Built{
		{ID: "disabled", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "disabled", Message: "should not appear"}},
		}},
		{ID: "enabled", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "enabled", Message: "should appear"}},
		}},
	}, nil)

	ctx := &pass.Context{}
	findings := pipe.run(ctx, nil)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (disabled analyzer skipped), got %d", len(findings))
	}
	if findings[0].AnalyzerID != "enabled" {
		t.Errorf("expected enabled, got %s", findings[0].AnalyzerID)
	}
}

func TestPipelineRunsInRegistrationOrder(t *testing.T) {
	t.Parallel()

	pipe := newPipeline(nil, []registry.Built{
		{ID: "ast", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "ast"}},
		}},
		{ID: "static", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "static"}},
		}},
		{ID: "arch", Analyzer: stubAnalyzer{
			findings: []finding.Finding{{AnalyzerID: "arch"}},
		}},
	}, nil)

	ctx := &pass.Context{}
	findings := pipe.run(ctx, nil)

	want := []string{"ast", "static", "arch"}
	if len(findings) != len(want) {
		t.Fatalf("expected %d findings, got %d: %+v", len(want), len(findings), findings)
	}
	for idx, id := range want {
		if findings[idx].AnalyzerID != id {
			t.Errorf(
				"findings[%d].AnalyzerID = %q, want %q (registration order violated)",
				idx,
				findings[idx].AnalyzerID,
				id,
			)
		}
	}
}

func TestPipelineAppliesOverrides(t *testing.T) {
	t.Parallel()

	cfg := toAnalysisConfig(&config.GovernanceConfig{
		Severities: map[string]string{"golangci-lint": "info"},
		Overrides: []pass.Override{{
			Files:     glob.CompileSet([]string{"web/test/**"}),
			Analyzers: map[string]pass.OverrideConfig{"file-size": {Severity: "warning"}},
		}},
	})
	pipe := newPipeline(cfg, []registry.Built{
		{ID: "golangci-lint", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{
					AnalyzerID: "golangci-lint/errcheck",
					File:       "main.go",
					Severity:   finding.SeverityError,
				},
			},
		}},
		{ID: "file-size", Analyzer: stubAnalyzer{
			findings: []finding.Finding{
				{AnalyzerID: "file-size", File: "web/test/a.dart", Severity: finding.SeverityError},
				{AnalyzerID: "file-size", File: "web/lib/b.dart", Severity: finding.SeverityError},
			},
		}},
	}, nil)

	findings := pipe.run(&pass.Context{}, nil)
	want := []finding.Severity{finding.SeverityInfo, finding.SeverityWarning, finding.SeverityError}
	for idx, severity := range want {
		if findings[idx].Severity != severity {
			t.Errorf("finding %d severity = %v, want %v", idx, findings[idx].Severity, severity)
		}
	}
}

func TestPipelineTransformsRunInOrderAfterEveryAnalyzer(t *testing.T) {
	t.Parallel()

	stamp := func(id string) Transform {
		return func(findings []finding.Finding) []finding.Finding {
			seen := fmt.Sprintf("%d seen", len(findings))
			return append(findings, finding.Finding{AnalyzerID: id, Message: seen})
		}
	}
	pipe := newPipeline(nil, []registry.Built{
		{ID: "a1", Analyzer: stubAnalyzer{findings: []finding.Finding{{AnalyzerID: "a1"}}}},
		{ID: "a2", Analyzer: stubAnalyzer{findings: []finding.Finding{{AnalyzerID: "a2"}}}},
	}, []Transform{stamp("first"), stamp("second")})

	findings := pipe.run(&pass.Context{}, nil)

	got := make([]string, len(findings))
	for index, finding := range findings {
		got[index] = finding.AnalyzerID + ":" + finding.Message
	}
	want := []string{"a1:", "a2:", "first:2 seen", "second:3 seen"}
	if !slices.Equal(got, want) {
		t.Fatalf("findings = %v, want %v", got, want)
	}
}

func TestPipelineSeedComesFirstAndIsTransformed(t *testing.T) {
	t.Parallel()

	pipe := newPipeline(nil, []registry.Built{
		{ID: "a1", Analyzer: stubAnalyzer{findings: []finding.Finding{{AnalyzerID: "a1"}}}},
	}, []Transform{func(findings []finding.Finding) []finding.Finding {
		return append(findings, finding.Finding{AnalyzerID: "seen", Line: len(findings)})
	}})

	findings := pipe.run(&pass.Context{}, []finding.Finding{{AnalyzerID: "seed"}})

	got := make([]string, len(findings))
	for index, finding := range findings {
		got[index] = finding.AnalyzerID
	}
	if !slices.Equal(got, []string{"seed", "a1", "seen"}) || findings[2].Line != 2 {
		t.Fatalf("findings = %+v, want seed, a1, then the transform seeing both", findings)
	}
}

func TestSessionTransformsDropPackageLevelVarAtSingleUseSite(t *testing.T) {
	t.Parallel()

	site := func(id string) finding.Finding {
		return finding.Finding{
			AnalyzerID: id,
			File:       "a.go",
			Line:       3,
			Symbol:     "once",
			Severity:   finding.SeverityWarning,
		}
	}
	pipe := newPipeline(nil, []registry.Built{
		{ID: singleuse.ID, Analyzer: stubAnalyzer{
			findings: []finding.Finding{site(singleuse.ID)},
		}},
		{ID: packagevar.ID, Analyzer: stubAnalyzer{
			findings: []finding.Finding{site(packagevar.ID)},
		}},
	}, (&Session{}).transforms())

	findings := pipe.run(&pass.Context{}, nil)

	if len(findings) != 1 || findings[0].AnalyzerID != singleuse.ID {
		t.Fatalf("findings = %+v, want the single-use finding alone", findings)
	}
}

func TestSessionTransformsAttachTheDescriptorRule(t *testing.T) {
	t.Parallel()

	index, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	testCases := []struct {
		analyzerID string
		file       string
		wantRule   string
	}{
		{identifier.ID, "a.go", "naming"},
		{variadicindex.ID, "a.go", "flag-arguments"},
		{complexity.ID, "a.go", "code-smells"},
		{importgroup.ID, "a.go", "imports"},
		{importgroup.ID, "lib/a.dart", "dart-style"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.analyzerID+" "+testCase.file, func(t *testing.T) {
			t.Parallel()
			sess := &Session{index: index}
			pipe := newPipeline(nil, []registry.Built{{
				ID: testCase.analyzerID,
				Analyzer: stubAnalyzer{findings: []finding.Finding{{
					AnalyzerID: testCase.analyzerID, File: testCase.file, Message: "m",
				}}},
			}}, sess.transforms())

			findings := pipe.run(&pass.Context{}, nil)

			if len(findings) != 1 || findings[0].RuleID != testCase.wantRule {
				t.Fatalf("findings = %+v, want RuleID %q", findings, testCase.wantRule)
			}
			if findings[0].RuleSummary == "" {
				t.Fatalf("rule %q attached no summary", testCase.wantRule)
			}
		})
	}
}

func TestOrderFindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input []finding.Finding
		want  []string
	}{
		{
			name: "severity first, errors before warnings before infos",
			input: []finding.Finding{
				{AnalyzerID: "info", Severity: finding.SeverityInfo, File: "b.go", Line: 5},
				{AnalyzerID: "error", Severity: finding.SeverityError, File: "a.go", Line: 10},
				{AnalyzerID: "warn", Severity: finding.SeverityWarning, File: "c.go", Line: 1},
			},
			want: []string{"error", "warn", "info"},
		},
		{
			name: "file then line within one severity",
			input: []finding.Finding{
				{AnalyzerID: "b5", Severity: finding.SeverityWarning, File: "b.go", Line: 5},
				{AnalyzerID: "a10", Severity: finding.SeverityWarning, File: "a.go", Line: 10},
				{AnalyzerID: "a3", Severity: finding.SeverityWarning, File: "a.go", Line: 3},
			},
			want: []string{"a3", "a10", "b5"},
		},
		{
			name: "equal sites keep their arrival order",
			input: []finding.Finding{
				{AnalyzerID: "a", Severity: finding.SeverityWarning, File: "a.go", Line: 9},
				{AnalyzerID: "b", Severity: finding.SeverityWarning, File: "a.go", Line: 2},
				{AnalyzerID: "c", Severity: finding.SeverityWarning, File: "a.go", Line: 2},
			},
			want: []string{"b", "c", "a"},
		},
		{
			name: "identical analyzer, site and message collapse",
			input: []finding.Finding{
				{AnalyzerID: "x", File: "a.go", Line: 1, Message: "dup"},
				{AnalyzerID: "x", File: "a.go", Line: 1, Message: "dup"},
				{AnalyzerID: "x", File: "a.go", Line: 1, Message: "dup"},
				{AnalyzerID: "y", File: "a.go", Line: 2, Message: "other"},
			},
			want: []string{"x", "y"},
		},
		{
			name: "same site but another message stays",
			input: []finding.Finding{
				{AnalyzerID: "x", File: "a.go", Line: 1, Message: "one"},
				{AnalyzerID: "x", File: "a.go", Line: 1, Message: "two"},
			},
			want: []string{"x", "x"},
		},
		{name: "empty input"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, finding := range orderFindings(slices.Clone(testCase.input)) {
				got = append(got, finding.AnalyzerID)
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("order = %v, want %v", got, testCase.want)
			}
		})
	}
}
