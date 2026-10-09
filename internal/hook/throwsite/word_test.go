package throwsite

import "testing"

func TestScan_WholeWordsAndCalls(t *testing.T) {
	t.Parallel()

	testCases := []scanCase{
		{"property key named throw", ".ts", "const a = { throw: 1 }\n", nil},
		{"key named throw after a space", ".ts", "const a = { throw : 1 }\n", nil},
		{"unicode identifier ending in throw", ".ts", "const \u00e9throw = 1\n", nil},
		{"unicode identifier starting with throw", ".ts", "const throw\u00e9 = 1\n", nil},
		{"promise reject with a space", ".ts", "return Promise.reject (err)\n", []int{1}},
		{"promise reject as a value", ".ts", "const a = Promise.reject\n", nil},
		{"dart future error", ".dart", "return Future.error(e);\n", []int{1}},
		{"dart future error with a space", ".dart", "return Future.error (e);\n", []int{1}},
		{"dart future error as a tearoff", ".dart", "final a = Future.error;\n", nil},
		{"dart throw after a semicolon", ".dart", "a();throw b;\n", []int{1}},
	}
	runScanCases(t, testCases)
}
