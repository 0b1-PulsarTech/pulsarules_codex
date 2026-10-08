package cleaner

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
)

func marksAt(lines ...int) []invisible.Mark {
	marks := make([]invisible.Mark, 0, len(lines))
	for _, line := range lines {
		marks = append(marks, invisible.Mark{Line: line})
	}
	return marks
}

func TestTally_Add(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		reports []Report
		want    Tally
	}{
		{"nothing", nil, Tally{}},
		{
			"acted and remaining count apart",
			[]Report{{Acted: marksAt(1, 2), Remaining: marksAt(3)}},
			Tally{Acted: 2, Remaining: 1, Removable: 1},
		},
		{
			"a skipped file's markers are unreadable",
			[]Report{{SkipReason: "x", Remaining: marksAt(1, 2)}, {Remaining: marksAt(5)}},
			Tally{Remaining: 1, Removable: 1, Unreadable: 2},
		},
		{
			"held markers are not removable",
			[]Report{{Remaining: marksAt(1, 2, 3), Held: marksAt(2)}},
			Tally{Remaining: 3, Held: 1, Removable: 2},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var tally Tally
			for _, report := range testCase.reports {
				tally.Add(report)
			}
			if tally != testCase.want {
				t.Errorf("tally = %+v, want %+v", tally, testCase.want)
			}
		})
	}
}

type lineCase struct {
	name  string
	tally Tally
	want  string
}

func runLineCases(t *testing.T, line func(Tally) string, testCases []lineCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := line(testCase.tally); got != testCase.want {
				t.Errorf("line = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestTally_ReportLine(t *testing.T) {
	t.Parallel()

	const tail = "in files clean cannot read: fix their syntax first"
	const held = "left where clean does not edit (string literals or possible code)"
	const rerun = "; re-run with --write to remove the removable ones"
	testCases := []lineCase{
		{"report", Tally{Remaining: 2, Removable: 2}, "2 marker(s) found" + rerun},
		{"nothing removable", Tally{Remaining: 2}, "2 marker(s) found"},
		{
			"markers left where clean does not edit",
			Tally{Remaining: 3, Removable: 1, Held: 2},
			"3 marker(s) found" + rerun + "; 2 " + held,
		},
		{
			"only held markers",
			Tally{Remaining: 2, Held: 2},
			"2 marker(s) found; 2 " + held,
		},
		{
			"unreadable files",
			Tally{Remaining: 1, Removable: 1, Unreadable: 3},
			"1 marker(s) found" + rerun + "; 3 more " + tail,
		},
	}
	runLineCases(t, Tally.ReportLine, testCases)
}

func TestTally_WriteLine(t *testing.T) {
	t.Parallel()

	const tail = "in files clean cannot read: fix their syntax first"
	const held = "left where clean does not edit (string literals or possible code)"
	testCases := []lineCase{
		{
			"write",
			Tally{Acted: 4, Remaining: 2},
			"removed 4 carrier(s); 2 marker(s) left for a human",
		},
		{
			"markers left where clean does not edit",
			Tally{Acted: 1, Remaining: 3, Held: 2},
			"removed 1 carrier(s); 3 marker(s) left for a human; " +
				"2 of them " + held,
		},
		{
			"unreadable files",
			Tally{Acted: 1, Remaining: 2, Unreadable: 3},
			"removed 1 carrier(s); 5 marker(s) left for a human; 3 of them " + tail,
		},
	}
	runLineCases(t, Tally.WriteLine, testCases)
}

func TestMarkerLines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		marks []invisible.Mark
		want  string
	}{
		{"one line", marksAt(3), "markers on line 3"},
		{"one line twice", marksAt(3, 3), "markers on line 3"},
		{"a line repeats", marksAt(3, 3, 5), "markers on lines 3, 5"},
		{
			"eight lines fit",
			marksAt(1, 2, 3, 4, 5, 6, 7, 8),
			"markers on lines 1, 2, 3, 4, 5, 6, 7, 8",
		},
		{
			"a ninth line is cut", marksAt(1, 2, 3, 4, 5, 6, 7, 8, 9),
			"markers on lines 1, 2, 3, 4, 5, 6, 7, 8, ...",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := markerLines(testCase.marks); got != testCase.want {
				t.Errorf("markerLines = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestReport_HeldNote(t *testing.T) {
	t.Parallel()

	const escape = "write it as an escape (\\uXXXX); clean leaves it"
	testCases := []struct {
		name string
		file string
		body string
		want string
	}{
		{"a string literal", "a.go", "var a = \"x\u200By\"\n", "in a string literal: " + escape},
		{
			"a raw string literal",
			"a.go",
			"var a = `x\u200By`\n",
			"in a raw string literal: move it into an interpreted string as an escape " +
				"(\\uXXXX); clean leaves it",
		},
		{
			"possible code",
			"a.md",
			"```\nx\u200By\n```\n",
			"in possible code: judge it by hand; clean leaves it",
		},
		{"a marker that is not held", "a.go", "var a = 1 // x\u200By\n", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			report, err := New(root).Inspect(seed(t, root, testCase.file, testCase.body))
			if err != nil || len(report.Remaining) != 1 {
				t.Fatalf("Inspect: %v, %d markers", err, len(report.Remaining))
			}
			if got := report.HeldNote(report.Remaining[0]); got != testCase.want {
				t.Errorf("HeldNote = %q, want %q", got, testCase.want)
			}
		})
	}
}
