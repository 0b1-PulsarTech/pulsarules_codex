package core

import "testing"

func TestCommentScanner_HTMLBlockSpansLines(t *testing.T) {
	t.Parallel()

	scanner := NewCommentScanner(CommentSyntax{Blocks: []BlockDelim{{Open: "<!--", Close: "-->"}}})
	testCases := []struct {
		line string
		want bool
	}{
		{"<!--", true},
		{"a license header", true},
		{"spanning lines", true},
		{"-->", true},
		{"<html>", false},
	}
	for _, testCase := range testCases {
		if got := scanner.IsCommentLine(testCase.line); got != testCase.want {
			t.Errorf("IsCommentLine(%q) = %v, want %v", testCase.line, got, testCase.want)
		}
	}
}

func TestCommentScanner_LuaBlockSpansLines(t *testing.T) {
	t.Parallel()

	scanner := NewCommentScanner(CommentSyntax{
		Line:   []string{"--"},
		Blocks: []BlockDelim{{Open: "--[[", Close: "]]"}},
	})
	testCases := []struct {
		line string
		want bool
	}{
		{"--[[", true},
		{"a multi-line lua comment", true},
		{"]]", true},
		{"local x = 1", false},
	}
	for _, testCase := range testCases {
		if got := scanner.IsCommentLine(testCase.line); got != testCase.want {
			t.Errorf("IsCommentLine(%q) = %v, want %v", testCase.line, got, testCase.want)
		}
	}
}

func TestCommentScanner_LuaLineDashNoBlock(t *testing.T) {
	t.Parallel()

	scanner := NewCommentScanner(CommentSyntax{
		Line:   []string{"--"},
		Blocks: []BlockDelim{{Open: "--[[", Close: "]]"}},
	})
	if !scanner.IsCommentLine("-- a plain line comment") {
		t.Error("expected a plain -- comment to be recognised as a comment line")
	}
	if scanner.IsCommentLine("local x = 1") {
		t.Error("expected code after the line comment to not be a comment line")
	}
}

func TestCommentScanner_CSSSelectorIsNotAComment(t *testing.T) {
	t.Parallel()

	syntax := CommentSyntax{
		Line: []string{"//"},
		Blocks: []BlockDelim{
			{Open: "/*", Close: "*/"},
			{Open: "<!--", Close: "-->"},
		},
	}

	testCases := []struct {
		name string
		line string
		want bool
	}{
		{"css universal selector", "* {", false},
		{"css nested selector", "  * > div {", false},
		{"js line comment", "// a comment", true},
		{"js block open", "/* a comment", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewCommentScanner(syntax)
			if got := scanner.IsCommentLine(testCase.line); got != testCase.want {
				t.Errorf("IsCommentLine(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}

func TestCommentScanner_GoBlockContinuation(t *testing.T) {
	t.Parallel()

	scanner := NewCommentScanner(CommentSyntax{
		Line:   []string{"//"},
		Blocks: []BlockDelim{{Open: "/*", Close: "*/"}},
	})
	testCases := []struct {
		line string
		want bool
	}{
		{"/* opens a block", true},
		{"* continuation", true},
		{"still open, no leading star", true},
		{"*/", true},
		{"func bar() {}", false},
	}
	for _, testCase := range testCases {
		if got := scanner.IsCommentLine(testCase.line); got != testCase.want {
			t.Errorf("IsCommentLine(%q) = %v, want %v", testCase.line, got, testCase.want)
		}
	}
}
