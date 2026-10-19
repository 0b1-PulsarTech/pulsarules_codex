package printable

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHeadTail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		fit    func(string, int) string
		text   string
		limit  int
		want   string
		stable bool
	}{
		{"fits raw", Head, "a\nb", 4, "a\nb", false},
		{"head", Head, "abcd", 3, "abc", true},
		{"tail", Tail, "abcd", 3, "bcd", true},
		{"head backs off a rune", Head, "ab\u00e9", 3, "ab", true},
		{"tail skips a rune fragment", Tail, "\u00e9bc", 3, "bc", true},
		{"zero limit head", Head, "abc", 0, "", true},
		{"negative limit tail", Tail, "abc", -5, "", true},
		{"invalid bytes head", Head, "\x80\x80\x80\x80", 8, `\x80\x80`, true},
		{"invalid bytes tail", Tail, "\x80\x80\x80\x80", 8, `\x80\x80`, true},
		{"escaped size counts", Head, "a\nb", 3, `a\n`, true},
		{"tail of escaped", Tail, "a\nb", 3, `\nb`, true},
		{"nul grows four times", Head, "\x00\x00", 4, `\x00`, true},
		{
			"nul head stops before a cut escape",
			Head,
			strings.Repeat("\x00", 5),
			9,
			`\x00\x00`,
			true,
		},
		{
			"nul tail stops before a cut escape",
			Tail,
			strings.Repeat("\x00", 5),
			9,
			`\x00\x00`,
			true,
		},
		{
			"one zero width escape of the tail",
			Tail,
			strings.Repeat("\u200b", 10),
			8,
			`\u200b`,
			true,
		},
		{
			"one zero width escape of the head",
			Head,
			strings.Repeat("\u200b", 10),
			8,
			`\u200b`,
			true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.fit(testCase.text, testCase.limit)
			if got != testCase.want {
				t.Errorf("fit = %q, want %q", got, testCase.want)
			}
			if testCase.stable && Line(got) != got {
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
