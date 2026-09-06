package language

import "testing"

func TestIsGoPackageDeclaration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		line string
		want bool
	}{
		{"package foo", true},
		{"  package foo", true},
		{"packagefoobar", false},
		{"// package foo", false},
		{"import \"fmt\"", false},
	}
	for _, testCase := range testCases {
		if got := isGoPackageDeclaration(testCase.line); got != testCase.want {
			t.Errorf("isGoPackageDeclaration(%q) = %v, want %v", testCase.line, got, testCase.want)
		}
	}
}
