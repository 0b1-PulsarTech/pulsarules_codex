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
		want     []commentPiece
		wantCode bool
	}{
		{"blank line", []string{""}, nil, false},
		{"code line", []string{"package a"}, nil, true},
		{"line comment", []string{"// note "}, []commentPiece{{Rest: "note"}}, false},
		{"hash comment", []string{"# note"}, []commentPiece{{Rest: "note"}}, false},
		{
			"one-line block",
			[]string{"/* note */"},
			[]commentPiece{{InBlock: true, Rest: "note */"}},
			false,
		},
		{"opener alone", []string{"/*"}, []commentPiece{{InBlock: true}}, false},
		{
			"block then line comment",
			[]string{"/* a */ // b"},
			[]commentPiece{{InBlock: true, Rest: "a */ // b"}, {Rest: "b"}},
			false,
		},
		{
			"block then code",
			[]string{"/* a */ package a"},
			[]commentPiece{{InBlock: true, Rest: "a */ package a"}},
			true,
		},
		{
			"continued block",
			[]string{"/*", " * a", " */ // b"},
			[]commentPiece{{InBlock: true, Rest: "*/ // b"}, {Rest: "b"}},
			false,
		},
		{
			"lua block before line prefix",
			[]string{"--[[ a"},
			[]commentPiece{{InBlock: true, Rest: "a"}},
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scanner := NewCommentScanner(AnyCommentSyntax())
			var got []commentPiece
			var hasCode bool
			for _, line := range testCase.lines {
				got, hasCode = scanner.scan(line)
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

func TestCommentScanner_Body(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		syntax CommentSyntax
		lines  []string
		want   []string
		wantIn []bool
	}{
		{
			name: "ts continuation keeps slashes as text", syntax: DialectTS.Syntax(),
			lines:  []string{"/**", " * a", " //", " /*", " */", "// b"},
			want:   []string{"", "a", "//", "/*", "", "b"},
			wantIn: []bool{false, true, true, true, true, false},
		},
		{
			name: "go continuation keeps stars", syntax: DialectGo.Syntax(),
			lines:  []string{"/*", "* a", "*/"},
			want:   []string{"", "* a", ""},
			wantIn: []bool{false, true, true},
		},
		{
			name: "one-line block never continues", syntax: DialectGo.Syntax(),
			lines:  []string{"/* a */", "// b"},
			want:   []string{"a", "b"},
			wantIn: []bool{false, false},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := NewCommentScanner(testCase.syntax)
			for idx, line := range testCase.lines {
				body, inBlock := scan.Body(line)
				if body != testCase.want[idx] || inBlock != testCase.wantIn[idx] {
					t.Fatalf("Body(%q) = (%q, %v), want (%q, %v)",
						line, body, inBlock, testCase.want[idx], testCase.wantIn[idx])
				}
			}
		})
	}
}

func TestCommentScanner_NestedBlock(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		nested bool
		lines  []string
		want   []bool
	}{
		{
			"nested keeps the outer block open", true,
			[]string{"/* one", " /* two */", " three", " four */", "x"},
			[]bool{true, true, true, true, false},
		},
		{
			"flat block ends at the first closer", false,
			[]string{"/* one", " /* two */", " three", " four */", "x"},
			[]bool{true, true, false, false, false},
		},
		{
			"depth two carries across lines", true,
			[]string{"/* a /* b", "*/", "still */", "x"},
			[]bool{true, true, true, false},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scanner := NewCommentScanner(CommentSyntax{
				Blocks: []BlockDelim{{Open: "/*", Close: "*/", Nested: testCase.nested}},
			})
			got := make([]bool, 0, len(testCase.lines))
			for _, line := range testCase.lines {
				got = append(got, scanner.IsCommentLine(line))
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("IsCommentLine over %q = %v, want %v", testCase.lines, got, testCase.want)
			}
		})
	}
}
