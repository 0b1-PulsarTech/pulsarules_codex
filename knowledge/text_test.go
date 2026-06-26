package knowledge

import "testing"

func TestFirstSentence(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"single sentence", "One thing. And more.", "One thing."},
		{"no period", "no period here", "no period here"},
		{"empty", "", ""},
		{"leading period", ". rest", "."},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := FirstSentence(testCase.input); got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
