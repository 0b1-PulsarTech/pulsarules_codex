package mark

import (
	"strings"
	"testing"
)

type scanCase struct {
	name      string
	src       string
	wantClass Class
	wantCount int
}

// The refuse-to-strip rows are the point of the table: a rewrite that guessed
// wrong on emoji glue or on a dash inside data would corrupt content silently.
var scanCases = []scanCase{
	{"zero width space", "a\u200Bb", ClassStrip, 1},
	{"soft hyphen", "co\u00ADoperate", ClassStrip, 1},
	{"right-to-left override is a Trojan Source carrier", "a\u202Eb", ClassStrip, 1},
	{"invisible times", "a\u2062b", ClassStrip, 1},
	{"no-break space folds to a space", "a\u00A0b", ClassSpace, 1},
	{"ideographic space", "a\u3000b", ClassSpace, 1},
	{"joiner between ASCII is a carrier", "a\u200Db", ClassStrip, 1},
	{"joiner between emoji is glue", "\U0001F468\u200D\U0001F469", ClassContextual, 1},
	{"variation selector after emoji", "\u2764\uFE0F", ClassContextual, 1},
	{"private use character", "a\uE000b", ClassContextual, 1},
	{"tag character", "a\U000E0041b", ClassContextual, 1},
	{"em dash is reported, never rewritten", "a\u2014b", ClassTypographic, 1},
	{"curly quote", "\u201Chi\u201D", ClassTypographic, 2},
	{"ellipsis", "wait\u2026", ClassTypographic, 1},
	{"plain ASCII is clean", "func main() { return }", ClassStrip, 0},
	{"joiner at offset zero has no left neighbour", "\u200Dab", ClassContextual, 1},
	{"joiner at end of input has no right neighbour", "ab\u200D", ClassContextual, 1},
}

func TestScan(t *testing.T) {
	t.Parallel()

	for _, testCase := range scanCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			marks := Scan(testCase.src)
			if len(marks) != testCase.wantCount {
				t.Fatalf("marks = %d, want %d: %+v", len(marks), testCase.wantCount, marks)
			}
			for _, mark := range marks {
				if mark.Class != testCase.wantClass {
					t.Errorf(
						"%q classified %d, want %d",
						string(mark.Rune),
						mark.Class,
						testCase.wantClass,
					)
				}
				if mark.Name == "" {
					t.Errorf("%q has no name", string(mark.Rune))
				}
			}
		})
	}
}

// TestClean_LeavesWhatItCannotJudge is the safety assertion: everything the
// scanner marks contextual or typographic must survive Clean byte for byte.
func TestClean_LeavesWhatItCannotJudge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
	}{
		{"emoji glue", "\U0001F468\u200D\U0001F469 ships"},
		{"variation selector", "\u2764\uFE0F ships"},
		{"em dash in prose", "one - two \u2014 three"},
		{"curly quotes in a literal", "s := \"\u201Cquoted\u201D\""},
		{"leading byte order mark", "\uFEFFpackage p"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, acted := Clean(testCase.src)
			if got != testCase.src {
				t.Errorf("Clean rewrote %q to %q", testCase.src, got)
			}
			if len(acted) != 0 {
				t.Errorf("Clean acted on %+v, want none", acted)
			}
		})
	}
}

func TestClean_RemovesCarriers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
		want string
	}{
		{"zero width space", "a\u200Bb", "ab"},
		{"soft hyphen", "co\u00ADop", "coop"},
		{"no-break space becomes a space", "a\u00A0b", "a b"},
		{"joiner between ASCII", "a\u200Db", "ab"},
		{"several at once", "a\u200Bb\u00A0c\u00ADd", "ab cd"},
		{"bidi override", "x\u202Ey", "xy"},
		{"nothing to do", "clean", "clean"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, acted := Clean(testCase.src)
			if got != testCase.want {
				t.Fatalf("Clean(%q) = %q, want %q", testCase.src, got, testCase.want)
			}
			if testCase.src != testCase.want && len(acted) == 0 {
				t.Error("Clean changed the text but reported no marks")
			}
		})
	}
}

// TestScan_ReportsLineNumbers pins the location data the hook and the analyzer
// both report; an offset alone would make every finding unactionable.
func TestScan_ReportsLineNumbers(t *testing.T) {
	t.Parallel()

	marks := Scan("one\ntwo\u200Bx\nthree\u00A0y\n")
	if len(marks) != 2 {
		t.Fatalf("marks = %d, want 2", len(marks))
	}
	if marks[0].Line != 2 || marks[1].Line != 3 {
		t.Errorf("lines = %d and %d, want 2 and 3", marks[0].Line, marks[1].Line)
	}
}

func TestClean_IsIdempotent(t *testing.T) {
	t.Parallel()

	once, _ := Clean("a\u200Bb\u00A0c\u00ADd\u202Ee")
	twice, acted := Clean(once)
	if twice != once {
		t.Errorf("second Clean changed %q to %q", once, twice)
	}
	if len(acted) != 0 {
		t.Errorf("second Clean acted on %+v", acted)
	}
	if strings.ContainsAny(once, "\u200B\u00AD\u202E") {
		t.Errorf("carriers survived: %q", once)
	}
}
