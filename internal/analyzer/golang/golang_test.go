package golang

import "testing"

func TestGoHandler(t *testing.T) {
	t.Parallel()

	h := New()
	if h.ID() != "go" {
		t.Errorf("ID: got %q, want go", h.ID())
	}
	exts := h.Extensions()
	if len(exts) != 1 || exts[0] != ".go" {
		t.Errorf("Extensions: got %v, want [.go]", exts)
	}
}

func TestIsCommentLine(t *testing.T) {
	t.Parallel()

	h := New()
	testCases := []struct {
		line string
		want bool
	}{
		{"// comment", true},
		{"  // indented", true},
		{"/* block */", true},
		{"* continuation", true},
		{"package foo", false},
		{"func bar() {}", false},
		{"", false},
	}
	for _, testCase := range testCases {
		if got := h.IsCommentLine(testCase.line); got != testCase.want {
			t.Errorf("IsCommentLine(%q): got %v, want %v", testCase.line, got, testCase.want)
		}
	}
}

func TestIsPackageDeclaration(t *testing.T) {
	t.Parallel()

	h := New()
	testCases := []struct {
		line string
		want bool
	}{
		{"package foo", true},
		{"  package foo", true},
		{"packagefoobar", false},
		{"// package foo", false},
		{"import \"fmt\"", false},
	}
	for _, testCase := range testCases {
		if got := h.IsPackageDeclaration(testCase.line); got != testCase.want {
			t.Errorf("IsPackageDeclaration(%q): got %v, want %v", testCase.line, got, testCase.want)
		}
	}
}
