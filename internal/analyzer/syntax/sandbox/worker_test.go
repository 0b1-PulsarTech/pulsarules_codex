package sandbox

import "testing"

func TestExitStatusString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		status exitStatus
		want   string
	}{
		{name: "code only", status: exitStatus{code: 3}, want: "exit 3"},
		{name: "signal", status: exitStatus{code: -1}, want: "exit -1"},
		{name: "stderr", status: exitStatus{code: 76, stderr: `"x"`}, want: `exit 76, stderr "x"`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.status.String(); got != testCase.want {
				t.Fatalf("string %q, want %q", got, testCase.want)
			}
		})
	}
}
