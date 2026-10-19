package commitmsg

import "testing"

func TestIsFooterLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		line string
		want bool
	}{
		{"co-authored-by", "Co-Authored-By: Name <a@b.c>", true},
		{"claude-session", "Claude-Session: https://example.com/x", true},
		{"breaking change", "BREAKING CHANGE: drops the flag", true},
		{"breaking-change", "BREAKING-CHANGE: drops the flag", true},
		{"lowercase co-authored-by", "Co-authored-by: X <x@y.z>", true},
		{"uppercase claude-session", "CLAUDE-SESSION: abc", true},
		{"lowercase breaking change stays prose", "breaking change: x", false},
		{"bare tool key", "Co-Authored-By", false},
		{"tool key without colon", "Co-Authored-By x <y>", false},
		{"closes", "Closes: #12", true},
		{"refs", "Refs: #42", true},
		{"closes without colon is prose", "Closes #12", false},
		{"refs without colon is prose", "Refs #42", false},
		{"bare key", "Refs", false},
		{"body line", "A plain body line.", false},
		{"key not at line start", " Refs: #42", false},
		{"empty", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := isFooterLine(testCase.line); got != testCase.want {
				t.Errorf("isFooterLine(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}
