package sfc

import "testing"

func TestTagEnd(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		text string
		from int
		end  int
		stop int
	}{
		{name: "plain", text: "<a href=x>", from: 2, end: 9, stop: 10},
		{name: "quoted gt", text: `<a title=">">`, from: 2, end: 12, stop: 13},
		{name: "single quoted gt", text: `<a title='>'>`, from: 2, end: 12, stop: 13},
		{name: "expression gt", text: "<a b={x > y}>", from: 2, end: 12, stop: 13},
		{name: "string brace in an expression", text: `<a b={"}"}>`, from: 2, end: 10, stop: 11},
		{name: "open quote", text: `<a title="x>`, from: 2, end: -1, stop: 12},
		{name: "apostrophe in a value", text: "<a title=it's>", from: 2, end: 13, stop: 14},
		{name: "quote after blanks", text: `<a title= ">">`, from: 2, end: 13, stop: 14},
		{name: "open expression fails", text: "<a b={x>", from: 2, end: badBrace, stop: 8},
		{name: "regex in an expression", text: "<a b={/'/.test(x)}>", from: 2, end: 18, stop: 19},
		{name: "stray less-than", text: "<a b<c>", from: 2, end: textTag, stop: 4},
		{name: "less-than in a quote", text: `<a b="<">`, from: 2, end: 8, stop: 9},
		{name: "less-than in an expression", text: "<a b={x < y}>", from: 2, end: 12, stop: 13},
		{name: "no end", text: "<a b", from: 2, end: -1, stop: 4},
		{name: "bad brace resumes", text: "<a b={x++ / 2}\n c>", from: 2, end: 17, stop: 18},
		{name: "bad brace, tag", text: "<a b={x++ / 2}\n<c>", from: 2, end: textTag, stop: 15},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, stop := (&markup{lower: testCase.text}).tagEnd(testCase.from)
			if got != testCase.end || stop != testCase.stop {
				t.Fatalf("tagEnd %d %d, want %d %d", got, stop, testCase.end, testCase.stop)
			}
		})
	}
}

func TestIsVoid(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want bool
	}{
		{name: "br", want: true},
		{name: "img", want: true},
		{name: "input", want: true},
		{name: "wbr", want: true},
		{name: "div"},
		{name: "svelte:head"},
		{name: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isVoid(testCase.name); got != testCase.want {
				t.Fatalf("isVoid %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestMarkupWanted(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		attrs []string
		want  []bool
	}{
		{name: "one of each", attrs: []string{"", " module"}, want: []bool{true, true}},
		{name: "second instance", attrs: []string{"", " lang=ts"}, want: []bool{true, false}},
		{
			name:  "second module",
			attrs: []string{` context="module"`, " module"},
			want:  []bool{true, false},
		},
		{
			name:  "data script",
			attrs: []string{` type="application/json"`, ""},
			want:  []bool{false, true},
		},
		{
			name:  "third script",
			attrs: []string{"", " module", ""},
			want:  []bool{true, true, false},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			scan := &markup{}
			for idx, attrs := range testCase.attrs {
				if got := scan.wanted(attrs); got != testCase.want[idx] {
					t.Fatalf("script %d wanted %v, want %v", idx, got, testCase.want[idx])
				}
			}
		})
	}
}

func TestEndsImplicitly(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		want bool
	}{
		{name: "p", want: true},
		{name: "li", want: true},
		{name: "td", want: true},
		{name: "rp", want: true},
		{name: "div"},
		{name: "ul"},
		{name: "table"},
		{name: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := endsImplicitly(testCase.name); got != testCase.want {
				t.Fatalf("endsImplicitly %v, want %v", got, testCase.want)
			}
		})
	}
}
