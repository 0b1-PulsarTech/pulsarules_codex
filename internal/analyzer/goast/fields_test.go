package goast

import "testing"

func TestFieldCount(t *testing.T) {
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

			got := FieldCount(firstFunc(t, parse(t, testCase.source)).Type.Params)
			if got != testCase.want {
				t.Fatalf("FieldCount = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestFieldCountNil(t *testing.T) {
	t.Parallel()

	fn := firstFunc(t, parse(t, "func run() {}"))
	if got := FieldCount(fn.Type.Results); got != 0 {
		t.Fatalf("FieldCount(nil results) = %d, want 0", got)
	}
}
