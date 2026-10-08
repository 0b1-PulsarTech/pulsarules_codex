package tsdoc

import (
	"slices"
	"testing"
)

func TestDocLines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		comment string
		want    []string
	}{
		{name: "one line", comment: "/** hello */", want: []string{"hello"}},
		{name: "star prefixed block", comment: "/**\n * a\n * b\n */", want: []string{"a", "b"}},
		{
			name:    "blank line inside is kept",
			comment: "/**\n * a\n *\n * b\n */",
			want:    []string{"a", "", "b"},
		},
		{name: "empty", comment: "/** */", want: []string{}},
		{name: "no star prefix", comment: "/**\n  a\n  b\n*/", want: []string{"a", "b"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := docLines(testCase.comment); !slices.Equal(got, testCase.want) {
				t.Fatalf("lines %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestCommentBody(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		comment string
		want    string
	}{
		{comment: "// note", want: "note"},
		{comment: "//   spaced  ", want: "spaced"},
		{comment: "/* block */", want: "block"},
		{comment: "/* * star */", want: "star"},
		{comment: "/** doc */", want: "doc"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.comment, func(t *testing.T) {
			t.Parallel()

			if got := commentBody(testCase.comment); got != testCase.want {
				t.Fatalf("body %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestIsDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		line string
		want bool
	}{
		{line: "eslint-disable-next-line no-console", want: true},
		{line: "@ts-expect-error", want: true},
		{line: "pulsarules: ignore", want: true},
		{line: "prettier-ignore", want: true},
		{line: "istanbul ignore next", want: true},
		{line: "svelte-ignore a11y", want: true},
		{line: "plain prose", want: false},
		{line: "", want: false},
		{line: "mentions eslint-disable midway", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.line, func(t *testing.T) {
			t.Parallel()

			if got := isDirective(testCase.line); got != testCase.want {
				t.Fatalf("directive %v, want %v", got, testCase.want)
			}
		})
	}
}

func filler(count int) []string {
	lines := make([]string, count)
	for idx := range lines {
		lines[idx] = "more"
	}
	lines[0] = "@deprecated"
	return lines
}

func TestIsDeprecatedOnly(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		prose []string
		own   []string
		want  bool
	}{
		{
			name:  "tag",
			prose: []string{"@deprecated x"},
			own:   []string{"@deprecated x"},
			want:  true,
		},
		{
			name:  "prefix",
			prose: []string{"Deprecated: x"},
			own:   []string{"Deprecated: x"},
			want:  true,
		},
		{
			name:  "at the line limit",
			prose: filler(maxDeprecatedLines),
			own:   filler(maxDeprecatedLines),
			want:  true,
		},
		{
			name:  "over the line limit",
			prose: filler(maxDeprecatedLines + 1),
			own:   filler(maxDeprecatedLines + 1),
		},
		{
			name:  "blank inside",
			prose: []string{"@deprecated", "a"},
			own:   []string{"@deprecated", "", "a"},
		},
		{name: "other prose", prose: []string{"Does a thing"}, own: []string{"Does a thing"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := isDeprecatedOnly(testCase.prose, testCase.own); got != testCase.want {
				t.Fatalf("deprecated %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsSimplification(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		prose []string
		want  bool
	}{
		{prose: []string{"simplification: no cache"}, want: true},
		{prose: []string{"Does a thing", "simplification: later"}},
		{prose: []string{"Simplification: wrong case"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.prose[0], func(t *testing.T) {
			t.Parallel()

			if got := isSimplification(testCase.prose); got != testCase.want {
				t.Fatalf("simplification %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsTypeTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		line string
		want bool
	}{
		{line: "@type {A}", want: true},
		{line: "@type", want: true},
		{line: "@type {A} the thing"},
		{line: "@type {A"},
		{line: "@returns {void}", want: true},
		{line: "@returns {void} nothing"},
		{line: "@satisfies {A}", want: true},
		{line: "@satisfies {A} extra"},
		{line: "@param {string} name", want: true},
		{line: "@param {string} [name]", want: true},
		{line: "@param name", want: true},
		{line: "@param {A"},
		{line: "@param {string}\tname\tprose"},
		{line: "@param {string} name the display name"},
		{line: "@param {string} name - the name"},
		{line: "@param {Record<string, {a: number}>} map", want: true},
		{line: "@property {string} name", want: true},
		{line: "@property {string} name the name"},
		{line: "@typedef {{a: string}} Shape", want: true},
		{line: "@typedef {Object} Shape the shape"},
		{line: "@template T", want: true},
		{line: "@template {Base} T", want: true},
		{line: "@template T the type"},
		{line: "@callback Handler", want: true},
		{line: "@callback Handler runs it"},
		{line: "@import {A} from './a'", want: true},
		{line: "@types are listed"},
		{line: "@see other"},
		{line: "@"},
		{line: "plain"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.line, func(t *testing.T) {
			t.Parallel()

			if got := isTypeTag(testCase.line); got != testCase.want {
				t.Fatalf("isTypeTag %v, want %v", got, testCase.want)
			}
		})
	}
}
