package tsdoc

import (
	"slices"
	"testing"
)

func TestIsJSDoc(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		comment string
		want    bool
	}{
		{comment: "/** doc */", want: true},
		{comment: "/***/", want: true},
		{comment: "/**/"},
		{comment: "/* block */"},
		{comment: "// line"},
		{comment: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.comment, func(t *testing.T) {
			t.Parallel()

			if got := IsJSDoc(testCase.comment); got != testCase.want {
				t.Fatalf("IsJSDoc = %v, want %v", got, testCase.want)
			}
		})
	}
}

type classifyCase struct {
	name   string
	doc    string
	others []string
	want   Shape
}

func classifyCases() []classifyCase {
	return slices.Concat(classifyCasesHead(), classifyCasesTail())
}

func classifyCasesHead() []classifyCase {
	return []classifyCase{
		{name: "prose", doc: "/** Does it. */", want: Shape{}},
		{name: "empty", doc: "/** */", want: Shape{Exempt: true}},
		{name: "directive only", doc: "/** eslint-disable */", want: Shape{Exempt: true}},
		{
			name: "directive beside prose",
			doc:  "/**\n * Prose.\n * @ts-ignore\n */",
			want: Shape{Mixed: true},
		},
		{
			name:   "adjacent line directive",
			doc:    "/** Prose. */",
			others: []string{"// eslint-disable-next-line"},
			want:   Shape{Mixed: true},
		},
		{
			name:   "adjacent plain comment",
			doc:    "/** Prose. */",
			others: []string{"// note"},
			want:   Shape{},
		},
		{
			name: "type tag only",
			doc:  "/** @type {import('./$types').PageData} */",
			want: Shape{Exempt: true},
		},
	}
}

func classifyCasesTail() []classifyCase {
	return []classifyCase{
		{
			name: "several type tags",
			doc:  "/**\n * @param {string} name\n * @returns {void}\n */",
			want: Shape{Exempt: true},
		},
		{
			name: "tag beside prose",
			doc:  "/**\n * Does it.\n * @returns {void}\n */",
			want: Shape{},
		},
		{name: "unlisted tag", doc: "/** @see other */", want: Shape{}},
		{
			name: "param with prose",
			doc:  "/** @param {string} name the display name */",
			want: Shape{},
		},
		{name: "returns with prose", doc: "/** @returns {void} nothing */", want: Shape{}},
		{name: "type with prose", doc: "/** @type {A} the thing */", want: Shape{}},
		{name: "tag-like word", doc: "/** @types are listed */", want: Shape{}},
		{name: "deprecated", doc: "/** @deprecated use x */", want: Shape{Exempt: true}},
		{
			name: "simplification",
			doc:  "/** simplification: no cache */",
			want: Shape{Exempt: true},
		},
		{
			name:   "adjacent directive on an empty doc",
			doc:    "/** */",
			others: []string{"// eslint-disable"},
			want:   Shape{Exempt: true},
		},
	}
}

func TestClassify(t *testing.T) {
	t.Parallel()

	testCases := classifyCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Classify(testCase.doc, testCase.others); got != testCase.want {
				t.Fatalf("shape %+v, want %+v", got, testCase.want)
			}
		})
	}
}
