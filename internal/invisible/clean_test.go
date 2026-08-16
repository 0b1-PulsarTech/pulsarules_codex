package invisible

import (
	"strings"
	"testing"
)

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
