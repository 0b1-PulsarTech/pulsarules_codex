package command

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestDropsNote(t *testing.T) {
	t.Parallel()

	infoA := diagnostic{code: "a", severity: core.SeverityInfo, file: "a.dart"}
	warnB := diagnostic{code: "b", severity: core.SeverityWarning, file: "b.dart"}
	errorC := diagnostic{code: "c", severity: core.SeverityError, file: "c.dart"}

	testCases := []struct {
		name      string
		items     []diagnostic
		wantCount int
		wantWorst core.Severity
		wantFirst string
	}{
		{"none", nil, 0, core.SeverityInfo, ""},
		{"one", []diagnostic{warnB}, 1, core.SeverityWarning, "b.dart"},
		{"a later error outranks", []diagnostic{infoA, errorC}, 2, core.SeverityError, "a.dart"},
		{
			"a later info keeps the worst",
			[]diagnostic{errorC, infoA},
			2,
			core.SeverityError,
			"c.dart",
		},
		{"first is kept", []diagnostic{infoA, warnB, errorC}, 3, core.SeverityError, "a.dart"},
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
			`2 diagnostics on paths outside the project or missing (first: "x.dart" c)`,
		},
		{
			"names count and first",
			drops{
				count: 3, worst: core.SeverityWarning,
				first: diagnostic{code: "lint_x", file: "/out/x.dart", line: 7},
			},
			`3 diagnostics on paths outside the project or missing ` +
				`(first: "/out/x.dart":7 lint_x)`,
		},
		{
			"invalid code is generic",
			drops{count: 1, first: diagnostic{code: "Bad Code", file: "x.dart"}},
			`1 diagnostic on a path outside the project or missing ` +
				`(first: "x.dart" diagnostic)`,
		},
		{
			"control bytes in the path are quoted",
			drops{count: 1, first: diagnostic{code: "c", file: "a\x1b[0m\n"}},
			`1 diagnostic on a path outside the project or missing ` +
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
