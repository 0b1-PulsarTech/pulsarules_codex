package vcs

import (
	"slices"
	"testing"
)

func TestParseStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  string
		want []Change
	}{
		{"empty status", "", nil},
		{
			"space",
			" M my dir/a b.go\x00",
			[]Change{{Path: "my dir/a b.go", Staging: ' ', Worktree: 'M', Extension: ".go"}},
		},
		{
			"tab",
			"?? a\tb.go\x00",
			[]Change{{Path: "a\tb.go", Staging: '?', Worktree: '?', Extension: ".go"}},
		},
		{
			"non-ascii",
			" M conf/\u00e9.yaml\x00",
			[]Change{{Path: "conf/\u00e9.yaml", Staging: ' ', Worktree: 'M', Extension: ".yaml"}},
		},
		{
			"quote",
			" M \"q\".go\x00",
			[]Change{{Path: "\"q\".go", Staging: ' ', Worktree: 'M', Extension: ".go"}},
		},
		{
			"arrow in name",
			"?? a -> b.go\x00",
			[]Change{{Path: "a -> b.go", Staging: '?', Worktree: '?', Extension: ".go"}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertParsed(t, testCase.raw, testCase.want)
		})
	}
}

func TestParseStatus_Pairs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  string
		want []Change
	}{
		{
			"rename is new then old",
			"R  new -> x.go\x00old -> y.go\x00 M z.go\x00",
			[]Change{
				{
					Path: "new -> x.go", OldPath: "old -> y.go", Staging: 'R', Worktree: ' ',
					Extension: ".go", Staged: true,
				},
				{Path: "z.go", Staging: ' ', Worktree: 'M', Extension: ".go"},
			},
		},
		{
			"R or C inside a path is not a pair",
			"?? README.md\x00?? Cfg.go\x00 M z.go\x00",
			[]Change{
				{Path: "README.md", Staging: '?', Worktree: '?', Extension: ".md"},
				{Path: "Cfg.go", Staging: '?', Worktree: '?', Extension: ".go"},
				{Path: "z.go", Staging: ' ', Worktree: 'M', Extension: ".go"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertParsed(t, testCase.raw, testCase.want)
		})
	}
}

func TestParseStatus_Variants(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		raw  string
		want []Change
	}{
		{
			"staged test file with upper-case extension",
			"A  x_test.go\x00M  Y.GO\x00",
			[]Change{
				{
					Path:      "x_test.go",
					Staging:   'A',
					Worktree:  ' ',
					Extension: ".go",
					Staged:    true,
				},
				{Path: "Y.GO", Staging: 'M', Worktree: ' ', Extension: ".go", Staged: true},
			},
		},
		{
			"copy in worktree column",
			" C copy.go\x00src.go\x00",
			[]Change{
				{
					Path:      "copy.go",
					OldPath:   "src.go",
					Staging:   ' ',
					Worktree:  'C',
					Extension: ".go",
				},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertParsed(t, testCase.raw, testCase.want)
		})
	}
}

func assertParsed(t *testing.T, raw string, want []Change) {
	t.Helper()
	got := parseStatus(raw).Changes
	if !slices.Equal(got, want) {
		t.Fatalf("changes = %+v, want %+v", got, want)
	}
}
