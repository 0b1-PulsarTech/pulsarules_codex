package glob

import "testing"

func TestHasBrace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		want    bool
	}{
		{"empty", "", false},
		{"plain", "a/b.go", false},
		{"opening brace", "a{b", true},
		{"closing brace", "a}b", true},
		{"alternatives", "{a,b}/x", true},
		{"brace in a class", "[{]", false},
		{"closing brace in a class", "x[a}]y", false},
		{"unterminated class hides a brace", "[a{", false},
		{"class closes at the first bracket", "[]{", true},
		{"escaped opening brace", `a\{b`, false},
		{"escaped closing brace", `a\}b`, false},
		{"escaped backslash does not escape", `a\\{b`, true},
		{"escaped bracket opens no class", `\[{`, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := HasBrace(testCase.pattern); got != testCase.want {
				t.Fatalf("HasBrace(%q) = %v, want %v", testCase.pattern, got, testCase.want)
			}
		})
	}
}

func TestStarCount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		want    int
	}{
		{"empty", "", 0},
		{"no star", "a/b.go", 0},
		{"lone star is free", "*", 0},
		{"doublestar is free", "a/**/b", 0},
		{"lone star segment is free", "a/*/b", 0},
		{"star in a name", "*.go", 1},
		{"two stars in a name", "*a*", 2},
		{"a star per segment", "a*/b*", 2},
		{"doublestar prefix is not free", "**a", 2},
		{"free segments beside a counted one", "**/*/a*b*", 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := StarCount(testCase.pattern); got != testCase.want {
				t.Fatalf("StarCount(%q) = %d, want %d", testCase.pattern, got, testCase.want)
			}
		})
	}
}

func TestWildcardWork(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		want    int
	}{
		{"empty", "", 0},
		{"plain", "abc", 0},
		{"star costs nothing", "a*b", 0},
		{"each question mark", "a?b?", 2},
		{"class with its brackets", "[abc]", 5},
		{"unterminated class", "[a", 2},
		{"escaped question mark", `\?`, 0},
		{"escaped bracket in a class", `[\]]`, 4},
		{"escape outside a class", `a\*b`, 0},
		{"mixed", "?[a]?", 5},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := WildcardWork(testCase.pattern); got != testCase.want {
				t.Fatalf("WildcardWork(%q) = %d, want %d", testCase.pattern, got, testCase.want)
			}
		})
	}
}

func TestStarTail(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		pattern string
		want    int
	}{
		{"empty", "", 0},
		{"no star", "abc", 0},
		{"lone star", "*", 0},
		{"doublestar", "a/**/b", 0},
		{"text after the star", "a*b", 1},
		{"star first", "*abc", 3},
		{"later stars are not counted", "a*b*c", 2},
		{"tails add up over segments", "*a/b*cd", 3},
		{"star at the end of a segment", "dir/*", 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := StarTail(testCase.pattern); got != testCase.want {
				t.Fatalf("StarTail(%q) = %d, want %d", testCase.pattern, got, testCase.want)
			}
		})
	}
}
