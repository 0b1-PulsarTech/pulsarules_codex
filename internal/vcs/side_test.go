package vcs

import "testing"

func TestSide_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		side Side
		want string
	}{
		{"head", Head, "HEAD"},
		{"staged", Staged, "staged"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.side.String(); got != testCase.want {
				t.Fatalf("String() = %q, want %q", got, testCase.want)
			}
		})
	}
}
