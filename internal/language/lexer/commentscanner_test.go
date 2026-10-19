package lexer

import (
	"slices"
	"testing"
)

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

func TestCommentScanner_InBlock(t *testing.T) {
	t.Parallel()

	syntax := CommentSyntax{
		Line:   []string{"//"},
		Blocks: []BlockDelim{{Open: "/*", Close: "*/"}},
	}
	testCases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"fresh scanner", nil, false},
		{"open block", []string{"/* eslint-disable"}, true},
		{"open block continued", []string{"/* a", "b"}, true},
		{"closed block", []string{"/* a", "*/"}, false},
		{"one-line block", []string{"/* a */"}, false},
		{"block closed then reopened", []string{"/* a */ /* b"}, true},
		{"line comment", []string{"// a"}, false},
		{"unsupported delimiter", []string{"<!-- svelte-ignore a"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewCommentScanner(syntax)
			for _, line := range testCase.lines {
				scanner.IsCommentLine(line)
			}
			if got := scanner.InBlock(); got != testCase.want {
				t.Errorf("InBlock() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestCommentScanner_Scan(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		lines    []string
		want     []CommentPiece
		wantCode bool
	}{
		{"blank line", []string{""}, nil, false},
		{"code line", []string{"package a"}, nil, true},
		{"line comment", []string{"// note "}, []CommentPiece{{Rest: "note"}}, false},
		{"hash comment", []string{"# note"}, []CommentPiece{{Rest: "note"}}, false},
		{
			"one-line block",
			[]string{"/* note */"},
			[]CommentPiece{{InBlock: true, Rest: "note */"}},
			false,
		},
		{"opener alone", []string{"/*"}, []CommentPiece{{InBlock: true}}, false},
		{
			"block then line comment",
			[]string{"/* a */ // b"},
			[]CommentPiece{{InBlock: true, Rest: "a */ // b"}, {Rest: "b"}},
			false,
		},
		{
			"block then code",
			[]string{"/* a */ package a"},
			[]CommentPiece{{InBlock: true, Rest: "a */ package a"}},
			true,
		},
		{
			"continued block",
			[]string{"/*", " * a", " */ // b"},
			[]CommentPiece{{InBlock: true, Rest: "*/ // b"}, {Rest: "b"}},
			false,
		},
		{
			"lua block before line prefix",
			[]string{"--[[ a"},
			[]CommentPiece{{InBlock: true, Rest: "a"}},
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scanner := NewCommentScanner(AnyCommentSyntax())
			var got []CommentPiece
			var hasCode bool
			for _, line := range testCase.lines {
				got, hasCode = scanner.Scan(line)
			}
			if !slices.Equal(got, testCase.want) || hasCode != testCase.wantCode {
				t.Fatalf(
					"Scan = (%+v, %v), want (%+v, %v)",
					got,
					hasCode,
					testCase.want,
					testCase.wantCode,
				)
			}
		})
	}
}
