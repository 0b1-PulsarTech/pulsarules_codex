package filesize

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func TestHardFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		errorLines int
		isTest     bool
		want       int
	}{
		{name: "production keeps the cap", errorLines: 290, want: 290},
		{name: "test file gets 2.6x the cap", errorLines: 290, isTest: true, want: 754},
		{name: "no cap stays no cap in tests", errorLines: 0, isTest: true, want: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			change := core.FileChange{Path: "foo.svelte", IsTest: testCase.isTest}
			got := newLineLimits(180, testCase.errorLines).hardFor(change)
			if got != testCase.want {
				t.Errorf("hard cap = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestErrorLines(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		params    core.ParamSet
		extension string
		want      int
	}{
		{name: "svelte defaults to 290", extension: ".svelte", want: 290},
		{name: "go has no default cap", extension: ".go", want: 0},
		{name: "dart has no default cap", extension: ".dart", want: 0},
		{
			name:      "explicit cap applies to any language",
			params:    core.ParamSet{ParamErrorLines: 400},
			extension: ".go",
			want:      400,
		},
		{
			name:      "explicit cap replaces the svelte default",
			params:    core.ParamSet{ParamErrorLines: 350},
			extension: ".svelte",
			want:      350,
		},
		{
			name:      "explicit zero disables the svelte default",
			params:    core.ParamSet{ParamErrorLines: 0},
			extension: ".svelte",
			want:      0,
		},
		{
			name:      "unrelated params leave the default",
			params:    core.ParamSet{ParamMaxLines: 150},
			extension: ".svelte",
			want:      290,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := errorLines(testCase.params, testCase.extension)
			if got != testCase.want {
				t.Errorf("errorLines = %d, want %d", got, testCase.want)
			}
		})
	}
}
