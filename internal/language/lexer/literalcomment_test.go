package lexer

import "testing"

func TestLiteralScan_CommentLen(t *testing.T) {
	t.Parallel()

	comments := CommentSyntax{
		Line: []string{"--", "//"},
		Blocks: []BlockDelim{
			{Open: "--[[", Close: "]]"},
			{Open: "/*", Close: "*/"},
			{Open: "{-", Close: "-}", Nested: true},
		},
	}
	testCases := []struct {
		name   string
		rest   string
		want   int
		cutOff bool
	}{
		{name: "not a comment", rest: "x := 1", want: 0},
		{name: "closed block", rest: "/* a */ b", want: 7},
		{name: "block opener wins over its line prefix", rest: "--[[ a ]] b", want: 9},
		{name: "line prefix without a block", rest: "-- a\nb", want: 4},
		{name: "line comment to EOF", rest: "// a", want: 4},
		{name: "unclosed block runs to EOF", rest: "/* a\nb", want: 6, cutOff: true},
		{name: "plain block does not nest", rest: "/* a /* b */ c */", want: 12},
		{name: "nested block closes once balanced", rest: "{- a {- b -} c -} d", want: 17},
		{name: "nested block, deep", rest: "{- {- {- -} -} -}x", want: 17},
		{name: "nested block left open", rest: "{- a {- b -} c", want: 14, cutOff: true},
		{name: "nested close alone ends it", rest: "{- a -} b -}", want: 7},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scan := &literalScan{comments: comments}
			if got := scan.commentLen(testCase.rest); got != testCase.want {
				t.Errorf("commentLen(%q) = %d, want %d", testCase.rest, got, testCase.want)
			}
			if scan.cutOff != testCase.cutOff {
				t.Errorf("cutOff = %v, want %v", scan.cutOff, testCase.cutOff)
			}
		})
	}
}
