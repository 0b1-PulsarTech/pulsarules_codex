package gosyntax

import "testing"

func TestCyclomaticComplexity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		source string
		want   int
	}{
		{name: "no branches", source: "package foo\nfunc f() { _ = 1 }\n", want: 1},
		{name: "one if", source: "package foo\nfunc f() { if true {} }\n", want: 2},
		{
			name:   "for loop",
			source: "package foo\nfunc f() { for i := 0; i < 10; i++ {} }\n",
			want:   2,
		},
		{
			name:   "range loop",
			source: "package foo\nfunc f() { for range []int{} {} }\n",
			want:   2,
		},
		{
			name:   "logical operators",
			source: "package foo\nfunc f(a, b bool) { _ = a && b || a }\n",
			want:   3,
		},
		{
			name: "default case is free, select case counts",
			source: "package foo\nfunc f(ok bool) {\n\tswitch {\n\tcase ok:\n\tdefault:\n\t}\n" +
				"\tselect {\n\tcase <-make(chan int):\n\t}\n}\n",
			want: 3,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := cyclomaticComplexity(firstFunc(t, testCase.source))
			if got != testCase.want {
				t.Fatalf("got complexity %d, want %d", got, testCase.want)
			}
		})
	}
}
