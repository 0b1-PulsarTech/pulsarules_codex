package tsproc

import "testing"

func TestChild(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		args     []string
		handled  bool
		wantExit []int
	}{
		{name: "not a child", args: []string{"exe", "validate"}},
		{
			name: "no ceiling", args: []string{"exe", ChildArg}, handled: true,
			wantExit: []int{ExitProtocol},
		},
		{
			name: "bad ceiling", args: []string{"exe", ChildArg, "x"}, handled: true,
			wantExit: []int{ExitProtocol},
		},
		{
			name: "negative ceiling", args: []string{"exe", ChildArg, "-1"}, handled: true,
			wantExit: []int{ExitProtocol},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var exits []int
			handled := Child(testCase.args, echo, func(code int) { exits = append(exits, code) })
			if handled != testCase.handled || len(exits) != len(testCase.wantExit) {
				t.Fatalf("handled %v exits %v", handled, exits)
			}
		})
	}
}
