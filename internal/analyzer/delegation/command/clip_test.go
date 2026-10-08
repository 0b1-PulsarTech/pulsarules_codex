package command

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

const wantFloor = 4000

func TestFit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		fit    func(string, int) string
		text   string
		limit  int
		want   string
		stable bool
	}{
		{"fits raw", fitHead, "a\nb", 4, "a\nb", false},
		{"head", fitHead, "abcd", 3, "abc", true},
		{"tail", fitTail, "abcd", 3, "bcd", true},
		{"head backs off a rune", fitHead, "ab\u00e9", 3, "ab", true},
		{"tail skips a rune fragment", fitTail, "\u00e9bc", 3, "bc", true},
		{"zero limit head", fitHead, "abc", 0, "", true},
		{"negative limit tail", fitTail, "abc", -5, "", true},
		{"invalid bytes head", fitHead, "\x80\x80\x80\x80", 8, `\x80\x80`, true},
		{"invalid bytes tail", fitTail, "\x80\x80\x80\x80", 8, `\x80\x80`, true},
		{"escaped size counts", fitHead, "a\nb", 3, `a\n`, true},
		{"tail of escaped", fitTail, "a\nb", 3, `\nb`, true},
		{"nul grows four times", fitHead, "\x00\x00", 4, `\x00`, true},
		{"nul head stops before a cut escape", fitHead, many("\x00", 5), 9, `\x00\x00`, true},
		{"nul tail stops before a cut escape", fitTail, many("\x00", 5), 9, `\x00\x00`, true},
		{"one zero width escape of the tail", fitTail, many("\u200b", 10), 8, `\u200b`, true},
		{"one zero width escape of the head", fitHead, many("\u200b", 10), 8, `\u200b`, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.fit(testCase.text, testCase.limit)
			if got != testCase.want {
				t.Errorf("fit = %q, want %q", got, testCase.want)
			}
			if testCase.stable && escape.Line(got) != got {
				t.Errorf("fit result %q changes under a second escape", got)
			}
			if !wholeEscapes(got) {
				t.Errorf("fit result %q holds a split escape", got)
			}
		})
	}
}

func wholeEscapes(text string) bool {
	whole := regexp.MustCompile(`\\(x[0-9a-f]{2}|u[0-9a-f]{4}|U[0-9a-f]{8}|[abfnrtv"\\])`)
	return !strings.Contains(whole.ReplaceAllString(text, ""), `\`) && utf8.ValidString(text)
}

func renderedLine(finding core.Finding) string {
	line := "  [warning] " + escape.Line(finding.AnalyzerID) + ": " + escape.Line(finding.Message)
	if finding.File != "" {
		place := escape.Path(finding.File)
		if finding.Line > 0 {
			place += ":" + strconv.Itoa(finding.Line)
		}
		line += " (" + place + ")"
	}
	return line
}

func renderedArrow(finding core.Finding) string {
	return "    \u2192 " + escape.Line(finding.Suggestion)
}

type lineCase struct {
	name        string
	finding     core.Finding
	arrow       bool
	floor       int
	clipped     bool
	wantFile    string
	wantID      string
	wantMessage string
}

func many(text string, count int) string { return strings.Repeat(text, count) }

func located(item diagnostic, rel string) core.Finding {
	item.line = 1234567890
	return reports{id: "tool"}.diagnostic(item, rel)
}

func locationCases() []lineCase {
	wide := diagnostic{code: "c", message: many("m", 9000)}
	return []lineCase{
		{
			name:    "100000 char valid code",
			finding: located(diagnostic{code: many("a", 100_000), message: "m"}, "f"),
			wantID:  "tool/diagnostic", wantFile: "f",
		},
		{
			name:    "100 KB relative path",
			finding: located(wide, many("p", 100_000)),
			floor:   wantFloor, wantID: "tool/c",
		},
		{
			name:    "nul path escapes past the location bound",
			finding: located(diagnostic{code: "c", message: "m"}, many("\x00", 200)),
			wantID:  "tool/c",
		},
		{
			name:    "path within the bound is kept",
			finding: located(wide, many("p", 500)),
			floor:   wantFloor, wantFile: many("p", 500), wantID: "tool/c",
		},
		{
			name:    "path of exactly 512 bytes is kept",
			finding: located(wide, many("p", 512)),
			floor:   wantFloor, wantFile: many("p", 512), wantID: "tool/c",
		},
		{
			name:    "path of 513 bytes is dropped",
			finding: located(wide, many("p", 513)),
			floor:   wantFloor, wantID: "tool/c",
		},
	}
}

func diagnosticCases() []lineCase {
	spaced := diagnostic{code: "c", message: "m", suggestion: many("\u200b", 3000)}
	return []lineCase{
		{
			name:    "3000 nul in a message",
			finding: located(diagnostic{code: "c", message: many("\x00", 3000)}, "f"),
			clipped: true, floor: wantFloor, wantFile: "f", wantID: "tool/c",
		},
		{
			name:    "3000 invalid bytes in a message",
			finding: located(diagnostic{code: "c", message: many("\x80", 3000)}, "f"),
			clipped: true, floor: wantFloor, wantFile: "f", wantID: "tool/c",
		},
		{
			name:    "3000 zero width spaces in a suggestion",
			finding: located(spaced, "f"),
			clipped: true, arrow: true, floor: wantFloor, wantFile: "f", wantID: "tool/c",
		},
		{
			name:    "long message keeps the code suffix",
			finding: located(diagnostic{code: "Bad", message: many("m", 9000)}, "f"),
			clipped: true, floor: wantFloor, wantFile: "f", wantID: "tool/diagnostic",
			wantMessage: `(code "Bad")`,
		},
		{
			name:     "long bad code is clipped in the suffix",
			finding:  located(diagnostic{code: many("B", 5000), message: "m"}, "f"),
			wantFile: "f", wantID: "tool/diagnostic",
			wantMessage: `m (code "` + many("B", 62) + `")`,
		},
		{
			name:     "clipped code never ends inside an escape",
			finding:  located(diagnostic{code: many("\u200b", 20), message: "m"}, "f"),
			wantFile: "f", wantID: "tool/diagnostic",
			wantMessage: `m (code "` + many(`\u200b`, 10) + `")`,
		},
	}
}

func subIDCases() []lineCase {
	rep := reports{id: "tool"}
	return []lineCase{
		{
			name:    "3000 newline lines of stderr",
			finding: rep.failed("dart exited with code 4", many("a\n", 3000)),
			clipped: true, floor: wantFloor, wantID: "tool/failed",
		},
		{
			name:    "3000 zero width spaces of stderr",
			finding: rep.failed("dart exited with code 4", many("\u200b", 3000)),
			clipped: true, floor: wantFloor, wantID: "tool/failed",
		},
		{
			name:    "100 KB executable name",
			finding: rep.failed(many("n", 100_000)+" did not run", many("e", 9000)+"LAST"),
			clipped: true, floor: wantFloor, wantID: "tool/failed",
			wantMessage: ": " + many("e", 3553) + "LAST",
		},
		{
			name:    "nul reason counts its escaped size",
			finding: rep.failed(many("\x00", 100), many("e", 9000)+"LAST"),
			floor:   wantFloor, wantID: "tool/failed",
			wantMessage: ": " + many("e", 3665) + "LAST",
		},
		{
			name:    "timeout of a long name",
			finding: rep.timeout(many("n", 100_000)),
			clipped: true, floor: wantFloor, wantID: "tool/timeout",
		},
		{
			name:    "config echoing a long value",
			finding: rep.config(many("\x00", 9000)),
			clipped: true, floor: wantFloor, wantID: "tool/config",
		},
	}
}

func maxLineCase() lineCase {
	rep := reports{id: many("i", 64)}
	item := diagnostic{code: "c", message: many("m", 9000), line: math.MaxInt}
	return lineCase{
		name:    "64 byte id, 500 byte path and the largest line number",
		finding: rep.diagnostic(item, many("p", 500)),
		clipped: true, floor: wantFloor, wantFile: many("p", 500), wantID: rep.id + "/c",
	}
}

func maxIDCases() []lineCase {
	rep := reports{id: many("i", 64)}
	wide := diagnostic{code: many("c", 5000), message: many("m", 9000)}
	return []lineCase{
		{
			name:    "64 byte id with a long diagnostic",
			finding: rep.diagnostic(wide, "f"),
			clipped: true, floor: wantFloor, wantFile: "f", wantID: rep.id + "/diagnostic",
		},
		{
			name:    "64 byte id with a long stderr",
			finding: rep.failed(many("n", 100_000), many("e", 9000)),
			clipped: true, floor: wantFloor, wantID: rep.id + "/failed",
		},
		{
			name:    "64 byte id timeout",
			finding: rep.timeout(many("n", 100_000)),
			clipped: true, floor: wantFloor, wantID: rep.id + "/timeout",
		},
		{
			name:    "64 byte id config",
			finding: rep.config(many("\x00", 9000)),
			clipped: true, floor: wantFloor, wantID: rep.id + "/config",
		},
		maxLineCase(),
	}
}

func checkLengths(t *testing.T, testCase lineCase) {
	t.Helper()
	const wantLineBytes = 4096

	lines := []string{renderedLine(testCase.finding)}
	if testCase.arrow {
		lines = []string{renderedArrow(testCase.finding)}
	}
	for _, rendered := range lines {
		if len(rendered) > wantLineBytes || len(rendered) < testCase.floor {
			t.Errorf("rendered line = %d bytes, want %d..%d",
				len(rendered), testCase.floor, wantLineBytes)
		}
	}
}

func TestRenderedLineBounds(t *testing.T) {
	t.Parallel()

	testCases := slices.Concat(
		locationCases(), diagnosticCases(), subIDCases(), maxIDCases(),
	)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			checkLengths(t, testCase)
			checkFields(t, testCase)
		})
	}
}

func checkFields(t *testing.T, testCase lineCase) {
	t.Helper()
	finding := testCase.finding
	if finding.AnalyzerID != testCase.wantID || finding.File != testCase.wantFile {
		t.Errorf("id %q file %q", finding.AnalyzerID, finding.File)
	}
	if finding.File == "" && finding.Line != 0 {
		t.Errorf("a dropped location kept line %d", finding.Line)
	}
	if !strings.HasSuffix(finding.Message, testCase.wantMessage) {
		t.Errorf("Message = %.100q...%.80q", finding.Message, testCase.wantMessage)
	}
	for _, text := range []string{finding.Message, finding.Suggestion} {
		if !utf8.ValidString(text) || testCase.clipped && escape.Line(text) != text {
			t.Errorf("text changes under a second escape (%d bytes)", len(text))
		}
	}
}

func TestTailStartVisitsOnlyTheTail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		text        string
		limit       int
		wantStart   int
		wantVisited int
	}{
		{"ascii", strings.Repeat("x", 1<<20), 100, 1<<20 - 100, 101},
		{"two byte runes", strings.Repeat("\u00e9", 1<<19), 100, 1<<20 - 100, 51},
		{"limit zero", strings.Repeat("x", 1<<20), 0, 1 << 20, 1},
		{"limit above the text", "short", 100, 0, 5},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			visited := 0
			size := func(unit string) int {
				visited++
				return len(unit)
			}

			if got := tailStart(testCase.text, testCase.limit, size); got != testCase.wantStart {
				t.Errorf("tailStart = %d, want %d", got, testCase.wantStart)
			}
			if visited > testCase.wantVisited {
				t.Errorf("sized %d units, want at most %d", visited, testCase.wantVisited)
			}
		})
	}
}

func TestClipSizing(t *testing.T) {
	t.Parallel()

	quotedBudget := `"` + strings.Repeat(`\"`, 31) + `"`
	testCases := []struct {
		name string
		clip func() string
		want string
	}{
		{
			"code clip counts the quoted size",
			func() string { return codeSuffix(strings.Repeat(`"`, 40)) },
			" (code " + quotedBudget + ")",
		},
		{
			"head clip counts the escaped size",
			func() string { return fitHead(strings.Repeat(`\`, 10), 5) },
			strings.Repeat(`\`, 5),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.clip(); got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
