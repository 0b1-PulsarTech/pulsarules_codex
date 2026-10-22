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

func TestShorten(t *testing.T) {
	t.Parallel()

	const bs, acc, ctl = `\`, "\u00e9", "\x01"
	testCases := []struct {
		name  string
		text  string
		limit int
		want  string
	}{
		{"head fits", `a\x01`, 5, `a\x01`},
		{"head keeps whole runes", acc + acc + acc, 6, acc + acc + acc},
		{"head cut", "abcdefg", 6, "abc..."},
		{"head escapes the kept part", "a" + ctl + ctl + ctl, 12, `a\x01\x01...`},
		{"head never splits a rune", "ab" + acc + acc + "x", 6, "ab..."},
		{"head literal backslash", "ab" + bs + "xyzw", 6, "ab..."},
		{"head cut escape start", "ab" + bs + "xyzwv", 7, "ab..."},
		{"head cut unicode start", "ab" + bs + "u00zwvv", 9, "ab..."},
		{"head backslash pairs", "a" + bs + bs + bs + "xyzwvv", 8, "a..."},
		{"head cut supplementary escape", "ab" + bs + "U000e0001zz", 10, "ab..."},
		{"head cut short unicode escape", "ab" + bs + "u00e9zzz", 10, "ab..."},
		{"head complete escape text", "a" + bs + "x41yzvv", 8, "a" + bs + "x41..."},
		{"head invalid bytes", "a\xff\xff\xff", 8, `a\xff...`},
		{"head backslash then rune", "x" + bs + acc + "abcd", 4, "x..."},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Shorten(testCase.text, testCase.limit); got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		text  string
		limit int
		kept  string
		whole bool
	}{
		{"stops before the limit", "ab\x01cd", 6, "ab\x01", false},
		{"keeps all", "ab", 2, "ab", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			kept, whole := Fit(testCase.text, testCase.limit)
			if kept != testCase.kept || whole != testCase.whole {
				t.Fatalf("Fit() = %q, %v, want %q, %v", kept, whole, testCase.kept, testCase.whole)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, text string
		limit      int
		want       string
	}{
		{"fits", "a/b.yaml", 8, "a/b.yaml"},
		{"keeps the tail", "dddd/.git/p.yaml", 12, ".../p.yaml"},
		{"keeps a quoted tail", "d\xffd/p\x01", 12, "...d/p\x01"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := ShortenPath(testCase.text, testCase.limit); got != testCase.want {
				t.Fatalf("ShortenPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}
