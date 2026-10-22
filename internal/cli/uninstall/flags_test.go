package uninstall

import (
	"flag"
	"slices"
	"testing"
)

func bindUninstall(t *testing.T, args ...string) *Command {
	t.Helper()

	cmd := &Command{}
	flagSet := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	cmd.Bind(flagSet)
	if err := flagSet.Parse(args); err != nil {
		t.Fatalf("Parse(%q): %v", args, err)
	}
	return cmd
}

func TestBind_ParsesFlags(t *testing.T) {
	t.Parallel()

	cmd := bindUninstall(t, "--project", "/tmp/repo", "--target", "opencode", "--keep-skills")
	if cmd.project != "/tmp/repo" || !cmd.keepSkills {
		t.Errorf("unexpected command: %+v", cmd)
	}
	if !slices.Contains(cmd.target, "opencode") {
		t.Errorf("target = %v, want to contain opencode", cmd.target)
	}
	if got := cmd.ProjectDir(); got != "/tmp/repo" {
		t.Errorf("ProjectDir() = %q, want /tmp/repo", got)
	}
}

func TestBind_Defaults(t *testing.T) {
	t.Parallel()

	cmd := bindUninstall(t, "--project", "/tmp/repo")
	if cmd.keepSkills {
		t.Error("keepSkills should default to false")
	}
	if cmd.hooksScope != "" {
		t.Errorf("hooksScope = %q, want empty (uninstall defaults to unwiring both files)",
			cmd.hooksScope)
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
			"repeats accumulate unsplit",
			[]string{"--target", "claude", "--target", "cursor,agents"},
			[]string{"claude", "cursor,agents"},
		},
		{"blank value is kept", []string{"--target", " "}, []string{" "}},
		{"lone comma is kept", []string{"--target", ","}, []string{","}},
		{
			"empty repeat is kept",
			[]string{"--target", "opencode", "--target", ""},
			[]string{"opencode", ""},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := bindUninstall(t, testCase.args...).target; !slices.Equal(got, testCase.want) {
				t.Errorf("target = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestExplicitTargets(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		target []string
		want   []string
	}{
		{name: "none passed", want: nil},
		{name: "single target", target: []string{"opencode"}, want: []string{"opencode"}},
		{
			name:   "duplicate deduped preserving order",
			target: []string{"opencode", "claude", "opencode"},
			want:   []string{"opencode", "claude"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd := &Command{target: testCase.target}
			if got := cmd.explicitTargets(); !slices.Equal(got, testCase.want) {
				t.Errorf("explicitTargets() = %v, want %v", got, testCase.want)
			}
		})
	}
}
