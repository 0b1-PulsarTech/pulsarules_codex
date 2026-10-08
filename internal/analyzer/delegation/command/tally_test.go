package command

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

func TestDropsNote(t *testing.T) {
	t.Parallel()

	infoA := diagnostic{code: "a", severity: finding.SeverityInfo, file: "a.dart"}
	warnB := diagnostic{code: "b", severity: finding.SeverityWarning, file: "b.dart"}
	errorC := diagnostic{code: "c", severity: finding.SeverityError, file: "c.dart"}

	testCases := []struct {
		name      string
		items     []diagnostic
		wantCount int
		wantWorst finding.Severity
		wantFirst string
	}{
		{"none", nil, 0, finding.SeverityInfo, ""},
		{"one", []diagnostic{warnB}, 1, finding.SeverityWarning, "b.dart"},
		{"a later error outranks", []diagnostic{infoA, errorC}, 2, finding.SeverityError, "a.dart"},
		{
			"a later info keeps the worst",
			[]diagnostic{errorC, infoA},
			2,
			finding.SeverityError,
			"c.dart",
		},
		{"first is kept", []diagnostic{infoA, warnB, errorC}, 3, finding.SeverityError, "a.dart"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var lost drops
			for _, item := range testCase.items {
				lost.note(item)
			}

			if lost.count != testCase.wantCount || lost.worst != testCase.wantWorst ||
				lost.first.file != testCase.wantFirst {
				t.Errorf("drops = %+v, want %d %v %q", lost, testCase.wantCount,
					testCase.wantWorst, testCase.wantFirst)
			}
		})
	}
}

func TestReportsDropped(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		lost drops
		want string
	}{
		{
			"first omits a zero line",
			drops{count: 2, first: diagnostic{code: "c", file: "x.dart"}},
			`2 diagnostics on paths outside the project (first: "x.dart" c)`,
		},
		{
			"names count and first",
			drops{
				count: 3, worst: finding.SeverityWarning,
				first: diagnostic{code: "lint_x", file: "/out/x.dart", line: 7},
			},
			`3 diagnostics on paths outside the project ` +
				`(first: "/out/x.dart":7 lint_x)`,
		},
		{
			"invalid code is generic",
			drops{count: 1, first: diagnostic{code: "Bad Code", file: "x.dart"}},
			`1 diagnostic on a path outside the project ` +
				`(first: "x.dart" diagnostic)`,
		},
		{
			"control bytes in the path are quoted",
			drops{count: 1, first: diagnostic{code: "c", file: "a\x1b[0m\n"}},
			`1 diagnostic on a path outside the project ` +
				`(first: "a\x1b[0m\n" c)`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := reports{id: "tool"}.dropped(testCase.lost)

			if got.Message != testCase.want || got.AnalyzerID != "tool/dropped" ||
				got.Severity != testCase.lost.worst {
				t.Errorf("finding = %q %q %v", got.AnalyzerID, got.Message, got.Severity)
			}
		})
	}
}
