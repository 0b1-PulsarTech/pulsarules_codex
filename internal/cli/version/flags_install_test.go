package version

import (
	"slices"
	"testing"
)

type listFlagCase struct {
	name string
	args []string
	want []string
}

func runListFlagCases(t *testing.T, pick func(*Options) []string, testCases []listFlagCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			opts, err := ParseArgs(testCase.args)
			if err != nil {
				t.Fatalf("ParseArgs(%q): %v", testCase.args, err)
			}
			got := pick(opts)
			if (got == nil) != (testCase.want == nil) || !slices.Equal(got, testCase.want) {
				t.Errorf("got %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestParseArgs_GitHooksIsLastWins(t *testing.T) {
	t.Parallel()

	runListFlagCases(t, func(opts *Options) []string { return opts.GitHooks }, []listFlagCase{
		{
			"tolerates spaces and blanks",
			[]string{"install", "--git-hooks", " commit-msg , , pre-push "},
			[]string{"commit-msg", "pre-push"},
		},
		{"empty means none", []string{"install", "--git-hooks", ""}, nil},
		{
			"last wins",
			[]string{"install", "--git-hooks", "pre-push", "--git-hooks", "commit-msg"},
			[]string{"commit-msg"},
		},
		{
			"bad first value is replaced",
			[]string{"install", "--git-hooks", "nope", "--git-hooks", "pre-push"},
			[]string{"pre-push"},
		},
		{
			"empty last value clears",
			[]string{"install", "--git-hooks", "pre-push", "--git-hooks", ""},
			nil,
		},
	})
}

func TestParseArgs_SkillsIsLastWins(t *testing.T) {
	t.Parallel()

	runListFlagCases(t, func(opts *Options) []string { return opts.Skills }, []listFlagCase{
		{"empty stays not given", []string{"install", "--skills", ""}, nil},
		{
			"splits on commas and drops empties",
			[]string{"install", "--skills", "a,,b"},
			[]string{"a", "b"},
		},
		{"blank is given but empty", []string{"install", "--skills", " "}, []string{}},
		{"omitted stays nil", []string{"install"}, nil},
		{"last wins", []string{"install", "--skills", "a", "--skills", "b"}, []string{"b"}},
		{"empty last is not given", []string{"install", "--skills", "a", "--skills", ""}, nil},
		{
			"blank last is given but empty",
			[]string{"install", "--skills", "a", "--skills", " "},
			[]string{},
		},
	})
}

func TestParseArgs_LanguagesIsLastWins(t *testing.T) {
	t.Parallel()

	runListFlagCases(t, func(opts *Options) []string { return opts.Languages }, []listFlagCase{
		{"splits", []string{"install", "--languages", "go, dart"}, []string{"go", "dart"}},
		{
			"last wins",
			[]string{"install", "--languages", "go", "--languages", "dart"},
			[]string{"dart"},
		},
		{
			"bad first value is replaced",
			[]string{"install", "--languages", "cobol", "--languages", "go"},
			[]string{"go"},
		},
		{
			"blank last is given but empty",
			[]string{"install", "--languages", "go", "--languages", " "},
			[]string{},
		},
	})
}

func TestParseArgs_TargetKeepsEveryOccurrenceRaw(t *testing.T) {
	t.Parallel()

	runListFlagCases(t, func(opts *Options) []string { return opts.Target }, []listFlagCase{
		{
			"repeats accumulate unsplit",
			[]string{"uninstall", "--target", "claude", "--target", "cursor,agents"},
			[]string{"claude", "cursor,agents"},
		},
		{
			"two names in one value",
			[]string{"install", "--target", "claude,opencode"},
			[]string{"claude,opencode"},
		},
		{"padding is kept", []string{"install", "--target", " opencode "}, []string{" opencode "}},
		{"empty value is kept", []string{"install", "--target", ""}, []string{""}},
		{"blank value is kept", []string{"uninstall", "--target", " "}, []string{" "}},
		{"lone comma is kept", []string{"uninstall", "--target", ","}, []string{","}},
		{"trailing comma is kept", []string{"install", "--target", "claude,"}, []string{"claude,"}},
		{
			"empty repeat is kept",
			[]string{"uninstall", "--target", "opencode", "--target", ""},
			[]string{"opencode", ""},
		},
	})
}
