package install

import (
	"flag"
	"slices"
	"testing"
)

func bindInstall(t *testing.T, args ...string) *Command {
	t.Helper()

	cmd := &Command{}
	flagSet := flag.NewFlagSet("install", flag.ContinueOnError)
	cmd.Bind(flagSet)
	if err := flagSet.Parse(args); err != nil {
		t.Fatalf("Parse(%q): %v", args, err)
	}
	return cmd
}

func TestBind_Defaults(t *testing.T) {
	t.Parallel()

	cmd := bindInstall(t, "--project", "/tmp/repo")
	if want := []string{"commit-msg", "pre-commit"}; !slices.Equal(cmd.gitHooks, want) {
		t.Errorf("default gitHooks = %q, want %q", cmd.gitHooks, want)
	}
	if cmd.hooksScope != "project" || cmd.noGitHooks || cmd.noHooks || cmd.printHooks {
		t.Errorf("unexpected defaults: %+v", cmd)
	}
	if got := cmd.ProjectDir(); got != "/tmp/repo" {
		t.Errorf("ProjectDir() = %q, want /tmp/repo", got)
	}
}

func TestBind_ParsesFlags(t *testing.T) {
	t.Parallel()

	cmd := bindInstall(t, "--project", "/tmp/repo", "--no-hooks", "--print-hooks",
		"--git-hooks", "commit-msg,pre-push", "--no-git-hooks")
	if !cmd.noHooks || !cmd.printHooks || !cmd.noGitHooks {
		t.Errorf("flags not parsed: %+v", cmd)
	}
	if want := []string{"commit-msg", "pre-push"}; !slices.Equal(cmd.gitHooks, want) {
		t.Errorf("gitHooks = %q, want %q", cmd.gitHooks, want)
	}
}

func TestBind_ListFlagsAreLastWins(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		pick func(*Command) []string
		want []string
	}{
		{
			"git-hooks tolerates spaces and blanks",
			[]string{"--git-hooks", " commit-msg , , pre-push "},
			func(cmd *Command) []string { return cmd.gitHooks },
			[]string{"commit-msg", "pre-push"},
		},
		{
			"git-hooks empty means none",
			[]string{"--git-hooks", ""},
			func(cmd *Command) []string { return cmd.gitHooks },
			nil,
		},
		{
			"git-hooks last wins",
			[]string{"--git-hooks", "pre-push", "--git-hooks", "commit-msg"},
			func(cmd *Command) []string { return cmd.gitHooks },
			[]string{"commit-msg"},
		},
		{
			"skills empty stays not given",
			[]string{"--skills", ""},
			func(cmd *Command) []string { return cmd.skills },
			nil,
		},
		{
			"skills split on commas and drop empties",
			[]string{"--skills", "a,,b"},
			func(cmd *Command) []string { return cmd.skills },
			[]string{"a", "b"},
		},
		{
			"skills blank last is given but empty",
			[]string{"--skills", "a", "--skills", " "},
			func(cmd *Command) []string { return cmd.skills },
			[]string{},
		},
		{
			"languages last wins",
			[]string{"--languages", "go", "--languages", "dart"},
			func(cmd *Command) []string { return cmd.languages },
			[]string{"dart"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.pick(bindInstall(t, testCase.args...))
			if (got == nil) != (testCase.want == nil) || !slices.Equal(got, testCase.want) {
				t.Errorf("got %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestBind_TargetKeepsEveryOccurrenceRaw(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want []string
	}{
		{
			"two names in one value",
			[]string{"--target", "claude,opencode"},
			[]string{"claude,opencode"},
		},
		{"padding is kept", []string{"--target", " opencode "}, []string{" opencode "}},
		{"empty value is kept", []string{"--target", ""}, []string{""}},
		{"trailing comma is kept", []string{"--target", "claude,"}, []string{"claude,"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := bindInstall(t, testCase.args...).target; !slices.Equal(got, testCase.want) {
				t.Errorf("target = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestTargets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		target []string
		want   []string
	}{
		{name: "none defaults to claude", want: []string{"claude"}},
		{
			name: "given targets win", target: []string{"cursor", "claude"},
			want: []string{"cursor", "claude"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := &Command{target: testCase.target}
			if got := cmd.targets(); !slices.Equal(got, testCase.want) {
				t.Errorf("targets() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestBind_TargetDropsExactRepeats(t *testing.T) {
	t.Parallel()

	got := bindInstall(
		t,
		"--target",
		"cursor",
		"--target",
		"claude",
		"--target",
		"cursor",
	).targets()
	if want := []string{"cursor", "claude"}; !slices.Equal(got, want) {
		t.Errorf("targets() = %v, want %v", got, want)
	}
}
