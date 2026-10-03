package printable

import "testing"

func TestPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{"newline", "a\nb.go", `"a\nb.go"`},
		{"tab", "a\tb.go", `"a\tb.go"`},
		{"carriage return", "a\rb.go", `"a\rb.go"`},
		{"delete byte", "a\x7fb.go", `"a\x7fb.go"`},
		{"c1 next line", "a\u0085b.go", `"a\u0085b.go"`},
		{"line separator", "a\u2028b.go", `"a\u2028b.go"`},
		{"bidi override", "a\u202eb.go", `"a\u202eb.go"`},
		{"byte order mark", "a\ufeffb.go", `"a\ufeffb.go"`},
		{"invalid utf-8", "a\xffb.go", `"a\xffb.go"`},
		{"quote", "a\"b.go", `"a\"b.go"`},
		{"backslash", "a\\b.go", `"a\\b.go"`},
		{"rename arrow", "a -> b.go", `"a -> b.go"`},
		{"trailing arrow", "a ->", `"a ->"`},
		{"leading arrow", "-> b", `"-> b"`},
		{"edge space", "a ", `"a "`},
		{"plain space", "a b.go", "a b.go"},
		{"plain non-ascii", "caf\u00e9.go", "caf\u00e9.go"},
		{"plain path", "internal/a.go", "internal/a.go"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Path(testCase.in); got != testCase.want {
				t.Fatalf("Path(%q) = %q, want %q", testCase.in, got, testCase.want)
			}
		})
	}
}

func TestLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   string
		want string
	}{
		{"newline", "a\nb", `a\nb`},
		{"tab", "a\tb", `a\tb`},
		{"carriage return", "a\rb", `a\rb`},
		{"delete byte", "a\x7fb", `a\x7fb`},
		{"c1 next line", "a\u0085b", `a\u0085b`},
		{"line separator", "a\u2028b", `a\u2028b`},
		{"bidi override", "a\u202eb", `a\u202eb`},
		{"byte order mark", "a\ufeffb", `a\ufeffb`},
		{"invalid utf-8", "a\xffb\xc3", `a\xffb\xc3`},
		{"quote unchanged", `say "hi"`, `say "hi"`},
		{"backslash unchanged", `a\b`, `a\b`},
		{"rename arrow unchanged", "a -> b", "a -> b"},
		{"plain space", "a b", "a b"},
		{"plain non-ascii", "caf\u00e9", "caf\u00e9"},
		{"empty", "", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Line(testCase.in); got != testCase.want {
				t.Fatalf("Line(%q) = %q, want %q", testCase.in, got, testCase.want)
			}
		})
	}
}
