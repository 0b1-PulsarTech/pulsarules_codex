package message

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func TestHasCanonicalEmoji(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{"regular", ":wrench: feat: Add", false},
		{"ghost on regular", ":ghost: feat: Ghost", false},
		{"canonical initial", ":ghost: Initial Commit", true},
		{"lowercase initial", ":ghost: Initial commit", true},
		{"wrong emoji initial", ":smile: Initial Commit", false},
		{"extra emoji initial", ":ghost::smile: Initial Commit", false},
		{"bare initial", "Initial Commit", false},
		{"canonical merge", ":volcano: Merge branch x", true},
		{"volcano then more", ":volcano::tea: Merge x", true},
		{"wrong emoji merge", ":smile: Merge branch x", false},
		{"bare merge", "Merge branch x", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := hasCanonicalEmoji(commitmsg.Parse(testCase.input)); got != testCase.want {
				t.Errorf("hasCanonicalEmoji(%q) = %v, want %v", testCase.input, got, testCase.want)
			}
		})
	}
}

func TestIsScopeChar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		char rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'7', true},
		{'-', true},
		{'_', true},
		{' ', false},
		{'/', false},
		{'é', false},
	}
	for _, testCase := range testCases {
		t.Run(string(testCase.char), func(t *testing.T) {
			t.Parallel()
			if got := isScopeChar(testCase.char); got != testCase.want {
				t.Errorf("isScopeChar(%q) = %v, want %v", testCase.char, got, testCase.want)
			}
		})
	}
}
