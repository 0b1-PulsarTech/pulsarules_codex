package complexity

import "testing"

// checkFileCasesMagicNumbers covers findMagicNumbers, including the tuned
// octal file-mode exclusion (see isOctalModeLiteral): the thing that should
// still be flagged (a plain decimal, even one divisible by eight) and the
// thing that should now not be (a 0o-prefixed permission literal).
func checkFileCasesMagicNumbers() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "magic number detected",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 42\n" +
				"	_ = x\n" +
				"}\n",
			expect: 1,
		},
		{
			name: "octal file mode literal not flagged as magic number",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 0o750\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "decimal multiple of eight still flagged as magic number",
			source: "package foo\n" +
				"func f() {\n" +
				"	x := 488\n" +
				"	_ = x\n" +
				"}\n",
			expect: 1,
		},
	}
}

func TestIsOctalModeLiteral(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		val  string
		want bool
	}{
		{"lowercase o prefix", "0o750", true},
		{"uppercase o prefix", "0O750", true},
		{"legacy octal form not matched", "0750", false},
		{"decimal not matched", "750", false},
		{"decimal divisible by eight not matched", "488", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := isOctalModeLiteral(testCase.val); got != testCase.want {
				t.Fatalf("isOctalModeLiteral(%q) = %v, want %v", testCase.val, got, testCase.want)
			}
		})
	}
}
