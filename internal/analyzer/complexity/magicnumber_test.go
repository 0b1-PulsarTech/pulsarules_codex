package complexity

import "testing"

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
		{
			name: "strconv ParseInt bit-size argument not flagged as magic number",
			source: "package foo\n" +
				"import \"strconv\"\n" +
				"func f() {\n" +
				"	x, _ := strconv.ParseInt(\"1\", 10, 64)\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "strconv ParseFloat bit-size argument not flagged as magic number",
			source: "package foo\n" +
				"import \"strconv\"\n" +
				"func f() {\n" +
				"	x, _ := strconv.ParseFloat(\"1\", 64)\n" +
				"	_ = x\n" +
				"}\n",
			expect: 0,
		},
		{
			name: "bare sixty-four in a business expression still flagged as magic number",
			source: "package foo\n" +
				"func f(x int) int {\n" +
				"	return x * 64\n" +
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
