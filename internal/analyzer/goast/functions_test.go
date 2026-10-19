package goast

import "testing"

func TestCyclomaticComplexity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{name: "straight line", source: "func run() {}", want: 1},
		{
			name:   "branches and logical operators",
			source: "func run(ok bool) {\n\tif ok && !ok {\n\t\tfor range 3 {\n\t\t}\n\t}\n}",
			want:   4,
		},
		{
			name: "default case is free, select case counts",
			source: "func run(ok bool) {\n\tswitch {\n\tcase ok:\n\tdefault:\n\t}\n" +
				"\tselect {\n\tcase <-make(chan int):\n\t}\n}",
			want: 3,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := CyclomaticComplexity(firstFunc(t, parse(t, testCase.source)))
			if got != testCase.want {
				t.Fatalf("CyclomaticComplexity = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestParamCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{name: "grouped names", source: "func run(a int, b, c string) {}", want: 3},
		{name: "unnamed", source: "func run(int, string) {}", want: 2},
		{name: "none", source: "func run() {}", want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := ParamCount(firstFunc(t, parse(t, testCase.source)).Type.Params)
			if got != testCase.want {
				t.Fatalf("ParamCount = %d, want %d", got, testCase.want)
			}
		})
	}
}
