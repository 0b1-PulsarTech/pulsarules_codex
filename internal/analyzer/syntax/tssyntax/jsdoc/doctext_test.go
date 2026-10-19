package jsdoc

import "testing"

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
