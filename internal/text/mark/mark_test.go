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

func scanCases() []scanCase {
	return []scanCase{
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
		{"joiner between DEL and ASCII is a carrier", "\x7f\u200Db", ClassStrip, 1},
		{"joiner before DEL is a carrier", "a\u200D\x7f", ClassStrip, 1},
	}
}

func TestScan(t *testing.T) {
	t.Parallel()

	for _, testCase := range scanCases() {
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

func BenchmarkScan(b *testing.B) {
	ascii := strings.Repeat("func main() { return nil } // plain source line\n", 200)
	mixed := strings.Repeat(
		"caf\u00E9 \u2014 na\u00EFve \u200B text \u201Cquoted\u201D \u00A0 done\n", 200,
	)
	for _, benchCase := range []struct{ name, src string }{
		{"ascii", ascii},
		{"mixed", mixed},
	} {
		b.Run(benchCase.name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				Scan(benchCase.src)
			}
		})
	}
}

func TestClassifyRange(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		char      rune
		wantClass Class
		wantName  string
		wantOK    bool
	}{
		{"language tag", 0xE0001, ClassContextual, "tag character", true},
		{"before language tag", 0xE0000, ClassStrip, "", false},
		{"after language tag", 0xE0002, ClassStrip, "", false},
		{"tag range start", 0xE0020, ClassContextual, "tag character", true},
		{"tag range end", 0xE007F, ClassContextual, "tag character", true},
		{"before tag range", 0xE001F, ClassStrip, "", false},
		{"after tag range", 0xE0080, ClassStrip, "", false},
		{"private use start", 0xE000, ClassContextual, "private use character", true},
		{"private use end", 0xF8FF, ClassContextual, "private use character", true},
		{"before private use", 0xDFFF, ClassStrip, "", false},
		{"after private use", 0xF900, ClassStrip, "", false},
		{"plane 15 private use start", 0xF0000, ClassContextual, "private use character", true},
		{"plane 15 private use end", 0xFFFFD, ClassContextual, "private use character", true},
		{"before plane 15", 0xEFFFF, ClassStrip, "", false},
		{"after plane 15", 0xFFFFE, ClassStrip, "", false},
		{"plane 16 private use start", 0x100000, ClassContextual, "private use character", true},
		{"plane 16 private use end", 0x10FFFD, ClassContextual, "private use character", true},
		{"between plane 15 and 16", 0xFFFFF, ClassStrip, "", false},
		{"after plane 16", 0x10FFFE, ClassStrip, "", false},
		{"variation selector start", 0xFE00, ClassContextual, "variation selector", true},
		{"variation selector", 0xFE0F, ClassContextual, "variation selector", true},
		{"before variation selector", 0xFDFF, ClassStrip, "", false},
		{"after variation selector", 0xFE10, ClassStrip, "", false},
		{"variation supplement start", 0xE0100, ClassContextual, "variation selector", true},
		{"variation supplement end", 0xE01EF, ClassContextual, "variation selector", true},
		{"before variation supplement", 0xE00FF, ClassStrip, "", false},
		{"after variation supplement", 0xE01F0, ClassStrip, "", false},
		{"unnamed format character", 0x0600, ClassStrip, "unnamed format character", true},
		{"plain letter is not a mark", 'a', ClassStrip, "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			class, name, ok := classifyRange(testCase.char)
			if class != testCase.wantClass || name != testCase.wantName || ok != testCase.wantOK {
				t.Errorf(
					"classifyRange(%U) = (%v, %q, %v), want (%v, %q, %v)",
					testCase.char, class, name, ok,
					testCase.wantClass, testCase.wantName, testCase.wantOK,
				)
			}
		})
	}
}
