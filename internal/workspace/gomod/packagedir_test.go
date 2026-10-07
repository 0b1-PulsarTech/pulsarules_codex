package gomod

import "testing"

func TestIsNonPackageDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "root", dir: ".", want: false},
		{name: "plain nested", dir: "internal/cli", want: false},
		{name: "hidden segment", dir: "internal/.cache/pkg", want: true},
		{name: "underscore segment", dir: "internal/_gen", want: true},
		{name: "testdata segment", dir: "internal/testdata/pkg", want: true},
		{name: "testdata as substring", dir: "internal/testdatas", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsPackageDir(testCase.dir); got != testCase.want {
				t.Errorf("IsNonPackageDir(%q) = %v, want %v", testCase.dir, got, testCase.want)
			}
		})
	}
}

func TestIsHiddenName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "dot", input: ".git", want: true},
		{name: "underscore", input: "_x.go", want: true},
		{name: "plain", input: "main.go", want: false},
		{name: "empty", input: "", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsHiddenName(testCase.input); got != testCase.want {
				t.Errorf("IsHiddenName(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}
