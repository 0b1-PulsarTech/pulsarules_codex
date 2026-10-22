package governance

import (
	"flag"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
)

func TestCommand_Bind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want Command
	}{
		{name: "defaults keep preset and config unset", want: Command{scope: "full"}},
		{
			name: "every flag",
			args: []string{
				"--project", "p", "--preset", "strict", "--config", "c.yaml", "--scope", "commit",
				"--golangci-config", "g.yml", "--typographic-severity", "info",
				"--branch-extra-types", "a,b", "--all-files", "--include-generated",
			},
			want: Command{
				project: "p", preset: "strict", config: "c.yaml", scope: "commit",
				golangciConfig: "g.yml", allFiles: true, includeGenerated: true,
				tuning: flags.Tuning{TypographicSeverity: "info", BranchExtraTypes: "a,b"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var cmd Command
			fs := flag.NewFlagSet("governance", flag.ContinueOnError)
			cmd.Bind(fs)
			if err := fs.Parse(testCase.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if cmd != testCase.want {
				t.Errorf("Command = %+v, want %+v", cmd, testCase.want)
			}
		})
	}
}

func TestCommand_ProjectDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		cmd  *Command
		want string
	}{
		{
			name: "the flag wins over the env",
			cmd:  &Command{project: "/explicit", env: flags.Environment{ProjectDir: "/from-env"}},
			want: "/explicit",
		},
		{
			name: "falls back to the env var",
			cmd:  New(flags.Environment{ProjectDir: "/from-env"}),
			want: "/from-env",
		},
		{
			name: "with neither it reports nothing rather than guessing",
			cmd:  New(flags.Environment{}),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.cmd.ProjectDir(); got != testCase.want {
				t.Errorf("ProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}
