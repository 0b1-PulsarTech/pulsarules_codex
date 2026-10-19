package gomod

import "testing"

func TestIsPackageDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "root", dir: ".", want: true},
		{name: "plain nested", dir: "internal/cli", want: true},
		{name: "hidden segment", dir: "internal/.cache/pkg", want: false},
		{name: "underscore segment", dir: "internal/_gen", want: false},
		{name: "testdata segment", dir: "internal/testdata/pkg", want: false},
		{name: "testdata as substring", dir: "internal/testdatas", want: true},
		{name: "vendor", dir: "vendor/pkg", want: true},
		{name: "nested vendor", dir: "sub/vendor/pkg", want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := IsPackageDir(testCase.dir); got != testCase.want {
				t.Errorf("IsPackageDir(%q) = %v, want %v", testCase.dir, got, testCase.want)
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
