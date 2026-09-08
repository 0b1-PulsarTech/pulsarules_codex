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

func TestIsSQLPackageDeclaration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want bool
	}{
		{"create schema", "CREATE SCHEMA app;", true},
		{"lowercase create schema", "create schema app;", true},
		{"indented create schema", "  CREATE SCHEMA app;", true},
		{"a select statement", "SELECT 1;", false},
		{"a comment", "-- CREATE SCHEMA app;", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := isSQLPackageDeclaration(testCase.line); got != testCase.want {
				t.Errorf(
					"isSQLPackageDeclaration(%q) = %v, want %v", testCase.line, got, testCase.want,
				)
			}
		})
	}
}
