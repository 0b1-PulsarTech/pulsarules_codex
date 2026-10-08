package packagevar

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func blankReportCases() []levelVarCase {
	const prelude = "package p\n\nfunc two() (int, int) { return 0, 0 }\n\n"
	return []levelVarCase{
		{
			name: "blank pair reported once",
			files: map[string]string{
				"a.go": prelude + "var _, _ = two()\n",
			},
			want: []string{"a.go:5:package-level var _"},
		},
		{
			name: "blank discard beside a named var reports only the name",
			files: map[string]string{
				"a.go": "package p\n\nfunc f() (int, error) { return 0, nil }\n\nvar v, _ = f()\n",
			},
			want: []string{"a.go:5:package-level var v"},
		},
		{
			name: "blank side effect beside a sentinel flags only the blank",
			files: map[string]string{
				"a.go": "package p\n\nimport \"errors\"\n\nfunc register() int { return 0 }\n\n" +
					"var _, ErrA = register(), errors.New(\"a\")\n",
			},
			want: []string{"a.go:7:package-level var _"},
		},
		{
			name: "blank assertion beside a sentinel exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"errors\"\n\ntype T struct{}\n\n" +
					"var _, ErrA = (*T)(nil), errors.New(\"a\")\n",
			},
		},
		{
			name: "second blank assertion beside a sentinel exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"errors\"\n\ntype T struct{}\n\n" +
					"var ErrA, _ = errors.New(\"a\"), (*T)(nil)\n",
			},
		},
		{
			name: "blank plain call flagged",
			files: map[string]string{
				"a.go": "package p\n\nfunc registerDriver() int { return 0 }\n\n" +
					"var _ = registerDriver()\n",
			},
			want: []string{"a.go:5:package-level var _"},
		},
	}
}

func levelVarDedupeCases() []levelVarCase {
	const single = "single-use-declaration"
	return []levelVarCase{
		{
			name:  "skipped when single-use-declaration already reported the name",
			files: map[string]string{"a.go": "package p\n\nvar once = 1\n"},
			prior: []finding.Finding{{
				AnalyzerID: single, File: "a.go", Line: 3,
				Symbol: "once",
			}},
		},
		{
			name:  "same line other name still reported",
			files: map[string]string{"a.go": "package p\n\nvar a, b = 1, 2\n"},
			prior: []finding.Finding{{
				AnalyzerID: single, File: "a.go", Line: 3,
				Symbol: "a",
			}},
			want: []string{"a.go:3:package-level var b"},
		},
		{
			name:  "grouped name line differs from var line",
			files: map[string]string{"a.go": "package p\n\nvar (\n\tx = 1\n\ty = 2\n)\n"},
			prior: []finding.Finding{{
				AnalyzerID: single, File: "a.go", Line: 5,
				Symbol: "y",
			}},
			want: []string{"a.go:4:package-level var x"},
		},
		{
			name:  "not skipped for another analyzer at the same line",
			files: map[string]string{"a.go": "package p\n\nvar once = 1\n"},
			prior: []finding.Finding{{AnalyzerID: "other", File: "a.go", Line: 3, Symbol: "once"}},
			want:  []string{"a.go:3:package-level var once"},
		},
		{
			name:  "not skipped for single-use-declaration in another file",
			files: map[string]string{"a.go": "package p\n\nvar once = 1\n"},
			prior: []finding.Finding{{
				AnalyzerID: single, File: "b.go", Line: 3,
				Symbol: "once",
			}},
			want: []string{"a.go:3:package-level var once"},
		},
	}
}

func TestScanReports(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(blankReportCases(), levelVarDedupeCases()))
}

func TestDedupeAgainstRealSingleUseDeclaration(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"a.go": "package p\n\nvar once = 1\n\nfunc f() int { return once }\n",
	}
	ctx := buildCtx(t, files, nil, nil)
	ctx.Findings = singleuse.NewAnalyzer().Analyze(ctx)
	if len(ctx.Findings) != 1 {
		t.Fatalf("single-use findings = %+v, want 1", ctx.Findings)
	}
	got := NewAnalyzer().Analyze(ctx)
	if len(got) != 0 {
		t.Fatalf("package-level-var findings = %+v, want none after dedupe", got)
	}
	if empty := NewAnalyzer().Analyze(buildCtx(t, files, nil, nil)); len(empty) != 1 {
		t.Fatalf("without the prior finding = %+v, want 1", empty)
	}
}

func TestBlankSuggestion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "package conversion names the typed spelling",
			src:  "package p\n\nimport \"x/pkg\"\n\ntype T struct{}\n\nvar _ = pkg.Iface(T{})\n",
			want: "var _ pkg.Iface = T{}",
		},
		{
			name: "plain call says call it from a constructor",
			src:  "package p\n\nfunc register() int { return 0 }\n\nvar _ = register()\n",
			want: "call it from a constructor or main",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := buildCtx(t, map[string]string{"a.go": testCase.src}, nil, nil)
			got := NewAnalyzer().Analyze(ctx)
			if len(got) != 1 || !strings.Contains(got[0].Suggestion, testCase.want) {
				t.Fatalf("findings = %+v, want suggestion containing %q", got, testCase.want)
			}
			if strings.Contains(got[0].Suggestion, "move it into") {
				t.Fatalf("suggestion = %q, want no move-into-function text", got[0].Suggestion)
			}
		})
	}
}
