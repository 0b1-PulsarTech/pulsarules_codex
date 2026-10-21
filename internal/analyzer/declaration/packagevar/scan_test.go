package packagevar

import (
	"strings"
	"testing"
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

func TestScanReports(t *testing.T) {
	t.Parallel()

	runCases(t, blankReportCases())
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

			ctx := buildCtx(t, map[string]string{"a.go": testCase.src}, nil)
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
