package invisible

import "testing"

func TestNewTables(t *testing.T) {
	t.Parallel()

	lookup := newTables()
	testCases := []struct {
		name  string
		names map[rune]string
		probe rune
		want  string
	}{
		{"strip", lookup.strip, '\u200B', "zero width space"},
		{"space", lookup.space, '\u00A0', "no-break space"},
		{"contextual", lookup.contextual, '\u200D', "zero width joiner"},
		{"typographic", lookup.typographic, '\u2014', "em dash"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.names[testCase.probe]; got != testCase.want {
				t.Errorf("name of %U = %q, want %q", testCase.probe, got, testCase.want)
			}
			for char, name := range testCase.names {
				if name == "" {
					t.Errorf("%U has an empty name", char)
				}
				if char < 0x80 {
					t.Errorf("%U is ASCII", char)
				}
			}
		})
	}
}

func TestTablesAreDisjoint(t *testing.T) {
	t.Parallel()

	lookup := newTables()
	owner := map[rune]string{}
	testCases := []struct {
		name  string
		names map[rune]string
	}{
		{"strip", lookup.strip},
		{"space", lookup.space},
		{"contextual", lookup.contextual},
		{"typographic", lookup.typographic},
	}
	for _, testCase := range testCases {
		for char := range testCase.names {
			if first, taken := owner[char]; taken {
				t.Errorf("%U is in both %s and %s", char, first, testCase.name)
			}
			owner[char] = testCase.name
		}
	}
}

func TestClassRemovable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		class Class
		want  bool
	}{
		{ClassStrip, true},
		{ClassSpace, true},
		{ClassContextual, false},
		{ClassTypographic, false},
	}
	for _, testCase := range testCases {
		if got := testCase.class.Removable(); got != testCase.want {
			t.Errorf("class %d Removable = %v, want %v", testCase.class, got, testCase.want)
		}
	}
}
