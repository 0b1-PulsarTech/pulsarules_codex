package character

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type settleCase struct {
	name  string
	items []hintedFinding
	want  []string
}

func item(line int, message, text string, caution int) hintedFinding {
	found := finding.Finding{Line: line, Message: message, Suggestion: text}
	return hintedFinding{finding: found, caution: caution}
}

func TestSettle(t *testing.T) {
	t.Parallel()

	testCases := []settleCase{
		{"none", nil, []string{}},
		{"one", []hintedFinding{item(1, "a", "x", plain)}, []string{"x"}},
		{
			"the most cautious hint wins for the line and message",
			[]hintedFinding{item(1, "a", "p", plain), item(1, "a", "l", literal)},
			[]string{"l", "l"},
		},
		{
			"order does not matter",
			[]hintedFinding{item(1, "a", "l", literal), item(1, "a", "p", plain)},
			[]string{"l", "l"},
		},
		{
			"unscanned beats plain",
			[]hintedFinding{item(1, "a", "p", plain), item(1, "a", "u", unscanned)},
			[]string{"u", "u"},
		},
		{
			"literal beats unscanned",
			[]hintedFinding{item(1, "a", "u", unscanned), item(1, "a", "l", literal)},
			[]string{"l", "l"},
		},
		{
			"another message keeps its own hint",
			[]hintedFinding{item(1, "a", "p", plain), item(1, "b", "l", literal)},
			[]string{"p", "l"},
		},
		{
			"another line keeps its own hint",
			[]hintedFinding{item(1, "a", "p", plain), item(2, "a", "l", literal)},
			[]string{"p", "l"},
		},
	}
	runSettleCases(t, testCases)
}

func runSettleCases(t *testing.T, testCases []settleCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := settle(testCase.items)
			if len(got) != len(testCase.want) {
				t.Fatalf("settle = %d findings, want %d", len(got), len(testCase.want))
			}
			for idx, found := range got {
				if want := testCase.want[idx]; found.Suggestion != want {
					t.Errorf("finding %d hint = %q, want %q", idx, found.Suggestion, want)
				}
			}
		})
	}
}

func TestHint_SameLineMarkersShareTheCautiousHint(t *testing.T) {
	t.Parallel()

	const (
		escape = "write it as the escape \\u00A0"
		ascii  = "write it as the escape \\u2026"
	)
	testCases := []struct {
		name string
		rel  string
		body string
		typo bool
		want string
	}{
		{"comment then literal", "a.mjs", "/* \u00A0 */ m = '\u00A0';\n", false, escape},
		{"literal then comment", "a.mjs", "m = '\u00A0'; /* \u00A0 */\n", false, escape},
		{"go literal then comment", "a.go", "var m = \"\u00A0\" // \u00A0\n", false, escape},
		{"markdown prose then code", "a.md", "a\u00A0 `\u00A0`\n", false, codeNote},
		{"markdown code then prose", "a.md", "`\u00A0` a\u00A0\n", false, codeNote},
		{"typographic pair", "a.mjs", "a = 1 /* \u2026 */ + '\u2026';\n", true, ascii},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := fileContext(t, testCase.rel, testCase.body)
			got := NewTextAnalyzer().Analyze(ctx)
			if testCase.typo {
				got = NewTypographicAnalyzer().Analyze(ctx)
			}
			if len(got) != 2 {
				t.Fatalf("findings = %d, want 2: %+v", len(got), got)
			}
			for _, found := range got {
				if found.Suggestion != testCase.want {
					t.Errorf("hint = %q, want %q", found.Suggestion, testCase.want)
				}
			}
		})
	}
}
