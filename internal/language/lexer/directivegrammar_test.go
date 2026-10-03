package lexer

import "testing"

func TestIsDirectiveHeaderLine_Grammar(t *testing.T) {
	t.Parallel()

	testCases := []headerTestCase{
		{name: "global identifier list", line: "/*global window*/", want: true},
		{name: "global with writable flag", line: "/* global a:writable, b */", want: true},
		{name: "globals spelling", line: "/* globals foo, bar */", want: true},
		{name: "global prose", line: "/* global counters live here. */", want: false},
		{name: "global whitespace separated", line: "/* global a b */", want: true},
		{name: "global trailing comma on a multi-line list", line: "/* global a,", want: true},
		{name: "globals dollar identifier", line: "/*globals $*/", want: true},
		{name: "global dangling colon item", line: "/* global :writable */", want: false},
		{name: "global empty list", line: "/* global */", want: false},
		{name: "eslint without a space", line: "/*eslint-disable*/", want: true},
		{name: "nolint bare", line: "//nolint", want: true},
		{name: "nolint linters", line: "//nolint:gosec,errcheck", want: true},
		{name: "nolint with reason", line: "//nolint:gosec // why", want: true},
		{name: "nolint bare with reason", line: "//nolint // why", want: true},
		{name: "nolint prose", line: "//nolint is never used here", want: false},
		{name: "nolint empty linter list", line: "//nolint:", want: false},
		{name: "nolint comma separated with space", line: "//nolint:errcheck, gosec", want: true},
		{name: "nolint space before comma", line: "//nolint:a ,b", want: true},
		{name: "nolint space after colon", line: "//nolint: errcheck", want: true},
		{name: "nolint two spaces before reason", line: "//nolint:errcheck  // why", want: true},
		{name: "nolint tab before reason", line: "//nolint:errcheck\t// why", want: true},
		{name: "nolint hyphenated linter", line: "//nolint:go-critic", want: true},
		{name: "nolint trailing comma", line: "//nolint:errcheck,", want: false},
		{name: "nolint linters then prose", line: "//nolint:gosec is fine", want: false},
		{name: "dart format width", line: "// dart format width=100", want: true},
		{name: "dart format off", line: "// dart format off", want: true},
		{name: "dart format on", line: "// dart format on", want: true},
		{name: "dart format prose", line: "// dart format is enforced by CI", want: false},
		{name: "dart format bad width", line: "// dart format width=wide", want: false},
		{name: "atlas directive", line: "-- atlas:txmode none", want: true},
		{name: "atlas prose", line: "-- atlas: we tried it", want: false},
		{name: "atlas bare", line: "-- atlas:", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := isDirectiveHeaderLine(testCase.line); got != testCase.want {
				t.Fatalf(
					"IsDirectiveHeaderLine(%q) = %v, want %v",
					testCase.line,
					got,
					testCase.want,
				)
			}
		})
	}
}
