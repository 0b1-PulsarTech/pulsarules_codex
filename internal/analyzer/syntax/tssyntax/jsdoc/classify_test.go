package jsdoc

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/docshape"
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

			if got := IsBlock(testCase.comment); got != testCase.want {
				t.Fatalf("IsJSDoc = %v, want %v", got, testCase.want)
			}
		})
	}
}

type classifyCase struct {
	name   string
	doc    string
	others []string
	want   docshape.Shape
}

func classifyCases() []classifyCase {
	return slices.Concat(classifyCasesHead(), classifyCasesTail())
}

func classifyCasesHead() []classifyCase {
	return []classifyCase{
		{name: "prose", doc: "/** Does it. */", want: docshape.Shape{}},
		{name: "empty", doc: "/** */", want: docshape.Shape{Exempt: true}},
		{name: "directive only", doc: "/** eslint-disable */", want: docshape.Shape{Exempt: true}},
		{
			name: "directive beside prose",
			doc:  "/**\n * Prose.\n * @ts-ignore\n */",
			want: docshape.Shape{Mixed: true},
		},
		{
			name:   "adjacent line directive",
			doc:    "/** Prose. */",
			others: []string{"// eslint-disable-next-line"},
			want:   docshape.Shape{Mixed: true},
		},
		{
			name:   "adjacent plain comment",
			doc:    "/** Prose. */",
			others: []string{"// note"},
			want:   docshape.Shape{},
		},
		{
			name: "type tag only",
			doc:  "/** @type {import('./$types').PageData} */",
			want: docshape.Shape{Exempt: true},
		},
	}
}

func classifyCasesTail() []classifyCase {
	return []classifyCase{
		{
			name: "several type tags",
			doc:  "/**\n * @param {string} name\n * @returns {void}\n */",
			want: docshape.Shape{Exempt: true},
		},
		{
			name: "tag beside prose",
			doc:  "/**\n * Does it.\n * @returns {void}\n */",
			want: docshape.Shape{},
		},
		{name: "unlisted tag", doc: "/** @see other */", want: docshape.Shape{}},
		{
			name: "param with prose",
			doc:  "/** @param {string} name the display name */",
			want: docshape.Shape{},
		},
		{name: "returns with prose", doc: "/** @returns {void} nothing */", want: docshape.Shape{}},
		{name: "type with prose", doc: "/** @type {A} the thing */", want: docshape.Shape{}},
		{name: "tag-like word", doc: "/** @types are listed */", want: docshape.Shape{}},
		{name: "deprecated", doc: "/** @deprecated use x */", want: docshape.Shape{Exempt: true}},
		{
			name: "simplification",
			doc:  "/** simplification: no cache */",
			want: docshape.Shape{Exempt: true},
		},
		{
			name:   "adjacent directive on an empty doc",
			doc:    "/** */",
			others: []string{"// eslint-disable"},
			want:   docshape.Shape{Exempt: true},
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
