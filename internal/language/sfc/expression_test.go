package sfc

import "testing"

func TestSkipBraces(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		from int
		want int
		fail bool
	}{
		{name: "simple", text: "{a}b", want: 3},
		{name: "nested", text: "{a{b}c}d", want: 7},
		{name: "string brace", text: `{"}"}x`, want: 5},
		{name: "single quoted brace", text: "{'}'}x", want: 5},
		{name: "template brace", text: "{`}`}x", want: 5},
		{name: "escaped quote", text: `{"\"}"}x`, want: 7},
		{name: "from the middle", text: "ab{c}d", from: 2, want: 5},
		{name: "regex with a quote", text: "{x.replace(/'/g, \"\")}y", want: 21},
		{name: "regex first", text: "{/'/.test(x)}y", want: 13},
		{name: "regex with a brace", text: "{x.test(/}/)}y", want: 13},
		{name: "sigils are plain text", text: "{#a}b", want: 4},
		{name: "unterminated fails", text: "{a", want: 2, fail: true},
		{name: "unterminated string fails", text: `{"a`, want: 3, fail: true},
		{name: "stray closer fails", text: "{a)}b", want: 3, fail: true},
		{name: "line ends a string", text: "{'a\n}b", want: 5},
		{name: "post-increment then a slash", text: "{x++ / 2}\n<p>", want: 9},
		{name: "misread regex resumes at its line end", text: "{a ? /b}\nc", want: 8, fail: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := skipBraces(testCase.text, testCase.from)
			if got.End != testCase.want || got.Closed == testCase.fail {
				t.Fatalf("skipBraces %+v, want %d, fail %v", got, testCase.want, testCase.fail)
			}
		})
	}
}

func TestSkipMustache(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		want int
		fail bool
	}{
		{name: "plain", text: "{a}b", want: 3},
		{name: "block open", text: "{#if a}b", want: 7},
		{name: "block close", text: "{/if}b", want: 5},
		{name: "slash then a comment", text: "{/* c } */ a}b", want: 13},
		{name: "slash then a line comment", text: "{// c\n a}b", want: 9},
		{name: "slash then a digit", text: "{/1/.test(a)}b", want: 13},
		{name: "decrement then a slash", text: "{x-- / 2}\nb", want: 9},
		{name: "branch", text: "{:else}b", want: 7},
		{name: "raw html", text: "{@html a}b", want: 9},
		{name: "blanks before the sigil", text: "{ #each a as b}c", want: 15},
		{name: "regex with a quote", text: "{x.replace(/'/g, \"\")}y", want: 21},
		{name: "blanks then a block with a regex", text: "{ #if /}/.test(a)}x", want: 18},
		{name: "block with a regex", text: "{#if /}/.test(a)}x", want: 17},
		{name: "unterminated fails", text: "{#if a", want: 6, fail: true},
		{name: "keyword then a regex", text: "{@html /}/.source}x", want: 18},
		{name: "only a sigil", text: "{#", want: 2, fail: true},
		{name: "only blanks", text: "{ ", want: 2, fail: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := skipMustache(testCase.text, 0)
			if got.End != testCase.want || got.Closed == testCase.fail {
				t.Fatalf("skipMustache %+v, want %d, fail %v", got, testCase.want, testCase.fail)
			}
		})
	}
}
