package sfc

import (
	"strings"
	"testing"
)

func TestTagName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		rest string
		want string
	}{
		{rest: "div>", want: "div"},
		{rest: "div class=a>", want: "div"},
		{rest: "br/>", want: "br"},
		{rest: "b<c>", want: "b"},
		{rest: "svelte:head\n>", want: "svelte:head"},
		{rest: "script", want: "script"},
		{rest: "", want: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.rest, func(t *testing.T) {
			t.Parallel()

			if got := tagName(testCase.rest); got != testCase.want {
				t.Fatalf("tagName %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestMarkupClosing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		open []string
		text string
		want []string
	}{
		{name: "top of the stack", open: []string{"a", "b"}, text: "</b>", want: []string{"a"}},
		{name: "pops unclosed", open: []string{"a", "b", "c"}, text: "</b>", want: []string{"a"}},
		{name: "nearest", open: []string{"a", "b", "a"}, text: "</a>", want: []string{"a", "b"}},
		{name: "unknown name", open: []string{"a"}, text: "</z>", want: []string{"a"}},
		{name: "empty stack", text: "</a>"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := &markup{lower: testCase.text, open: testCase.open}
			scan.closing()
			if len(scan.open) != len(testCase.want) {
				t.Fatalf("open %v, want %v", scan.open, testCase.want)
			}
			if scan.pos != len(testCase.text) {
				t.Fatalf("pos %d, want %d", scan.pos, len(testCase.text))
			}
		})
	}
}

func TestMarkupClosingWindow(t *testing.T) {
	t.Parallel()

	scan := &markup{lower: "</a>", open: []string{"a"}}
	for range 40 {
		scan.open = append(scan.open, "b")
	}
	scan.closing()
	if len(scan.open) != 41 {
		t.Fatalf("a match beyond the window popped the stack: %d open", len(scan.open))
	}
}

func TestMarkupExpression(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		text       string
		wantBlocks int
		wantPos    int
		wantStop   bool
	}{
		{name: "plain", text: "{a}b", wantPos: 3},
		{name: "block open", text: "{#if a}b", wantBlocks: 1, wantPos: 7},
		{name: "block open after blanks", text: "{ #each a as b}", wantBlocks: 1, wantPos: 15},
		{name: "block close", text: "{/if}", wantPos: 5},
		{name: "branch", text: "{:else}", wantPos: 7},
		{name: "unterminated stops the scan", text: "{#if a", wantPos: 6, wantStop: true},
		{name: "slash then a comment", text: "{/* c */ a}b", wantPos: 11},
		{name: "slash then a line comment", text: "{// c\n a}b", wantPos: 9},
		{name: "post-increment then a slash", text: "{n++ / 2}\nb", wantPos: 9},
		{name: "decrement then a slash", text: "{n-- / 2} }\nb", wantPos: 9},
		{name: "misread regex resumes at its line end", text: "{a ? /b}</p>\nc", wantPos: 12},
		{name: "block with a misread regex", text: "{#if a ? /b}\nc", wantBlocks: 1, wantPos: 12},
		{name: "stray closer resumes", text: "{a)}b", wantPos: 3},
		{name: "block close needs a name", text: "{/* c */}", wantPos: 9},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := &markup{lower: testCase.text}
			scan.expression()
			if scan.blocks != testCase.wantBlocks || scan.pos != testCase.wantPos ||
				scan.unclosed != testCase.wantStop {
				t.Fatalf("blocks %d pos %d unclosed %v", scan.blocks, scan.pos, scan.unclosed)
			}
		})
	}
}

func TestMarkupScanIsLinear(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		unit         string
		head         string
		tail         string
		wantUnclosed bool
	}{
		{name: "comment opener", unit: "/*{", wantUnclosed: true},
		{name: "line comment", unit: "//{", wantUnclosed: true},
		{name: "bare brace", unit: "{", wantUnclosed: true},
		{name: "post-increment", unit: "{n++ / 2}\n"},
		{name: "misread regex", unit: "{n ? /b}\n"},
		{name: "misread regex in a tag", unit: "<a b={n ? /b}\n"},
		{name: "misread regex with a block", unit: "{#if n ? /b}\n{/if}\n"},
		{name: "stray closer", unit: "{a)}"},
		{name: "stray angle", unit: "<a "},
		{name: "nested angles", unit: "<a <a "},
		{name: "open comment", unit: "<!-- "},
		{name: "open bang tag", unit: "<!x "},
		{name: "open closing tag", unit: "</a "},
		{name: "quoted attributes then a stray angle", unit: " y=\"<b\"", head: "<a", tail: " <"},
		{name: "single quoted attributes", unit: " y='<b'", head: "<a", tail: " <"},
		{name: "mustache attributes", unit: " y={x}", head: "<a", tail: " <"},
		{name: "open quote in a tag", unit: "<a b=\"{", wantUnclosed: true},
		{name: "unterminated string", unit: "{\"\n", wantUnclosed: true},
		{name: "deep brackets", unit: "{" + strings.Repeat("[", 49), wantUnclosed: true},
		{name: "unclosed raw tag", unit: "<style> "},
		{name: "plain markup", unit: "<p>a</p>\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			unclosed := scanLinear(t, testCase.head, testCase.unit, testCase.tail)
			if unclosed != testCase.wantUnclosed {
				t.Fatalf("unclosed %v, want %v", unclosed, testCase.wantUnclosed)
			}
		})
	}
}

func scanLinear(t *testing.T, head, unit, tail string) bool {
	t.Helper()

	var unclosed bool
	for _, size := range []int{1 << 10, 1 << 13, 1 << 16} {
		body := head + strings.Repeat(unit, size/len(unit))
		text := "<script>a</script>\n<p>" + body + tail
		scan := &markup{text: text, lower: text}
		if err := scan.run(); err != nil {
			t.Fatalf("run: %v", err)
		}
		if !linearWork(scan, len(text)) {
			t.Fatalf("size %d: scanned %d of %d bytes", size, scan.scanned, len(text))
		}
		unclosed = scan.unclosed
	}
	return unclosed
}

func linearWork(scan *markup, length int) bool {
	return scan.scanned <= 2*length && scan.scanned >= length/4
}
