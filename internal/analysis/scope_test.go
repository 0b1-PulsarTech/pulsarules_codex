package analysis

import "testing"

func TestParseScope(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  Scope
	}{
		{name: "commit spells ScopeCommit", input: "commit", want: ScopeCommit},
		{name: "changed spells ScopeChanged", input: "changed", want: ScopeChanged},
		{name: "full spells ScopeFull", input: "full", want: ScopeFull},
		{name: "empty defaults to ScopeFull", input: "", want: ScopeFull},
		{name: "unrecognized defaults to ScopeFull", input: "bogus", want: ScopeFull},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := ParseScope(testCase.input); got != testCase.want {
				t.Fatalf("ParseScope(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}
