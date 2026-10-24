package hookset

import "testing"

func TestShellQuote(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		value string
		want  string
	}{
		{"plain", "/home/user", `'/home/user'`},
		{"space", "/my dir", `'/my dir'`},
		{"single quote", "o'neil", `'o'\''neil'`},
		{"empty", "", `''`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := ShellQuote(testCase.value); got != testCase.want {
				t.Errorf("ShellQuote(%q) = %q, want %q", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestJoinShellList(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		separator rune
		want      string
	}{
		{"colon is bare", ':', `a:b`},
		{"semicolon is quoted", ';', `a";"b`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := JoinShellList(testCase.separator, "a", "b"); got != testCase.want {
				t.Errorf("JoinShellList = %q, want %q", got, testCase.want)
			}
		})
	}
}
