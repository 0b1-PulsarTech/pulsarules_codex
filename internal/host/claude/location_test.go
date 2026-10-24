package claude

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
)

func TestLocation_Command(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		loc  Location
		want string
	}{
		{
			name: "project",
			loc:  Location{},
			want: `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh" stop`,
		},
		{
			name: "global",
			loc:  Location{GlobalDir: "/home/user/.claude"},
			want: `bash '/home/user/.claude/hooks/skill-router-reminder.sh' stop`,
		},
		{
			name: "global path with a quote and a space",
			loc:  Location{GlobalDir: "/home/o'neil/my dir/.claude"},
			want: `bash '/home/o'\''neil/my dir/.claude/hooks/skill-router-reminder.sh' stop`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.loc.command("stop"); got != testCase.want {
				t.Errorf("command = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestLocation_ScriptValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		loc       Location
		separator rune
		want      hookset.ScriptValues
	}{
		{
			name:      "project lists its own skills, then the user's",
			loc:       Location{},
			separator: ':',
			want: hookset.ScriptValues{
				Bin: `"$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli"`,
				SkillsDirs: `"$CLAUDE_PROJECT_DIR/.claude/skills":` +
					`"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills"`,
				Log:         `"$CLAUDE_PROJECT_DIR/.claude/hook-execution.log"`,
				ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
			},
		},
		{
			name:      "a windows separator is quoted so the shell keeps one word",
			loc:       Location{},
			separator: ';',
			want: hookset.ScriptValues{
				Bin: `"$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli"`,
				SkillsDirs: `"$CLAUDE_PROJECT_DIR/.claude/skills"";"` +
					`"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills"`,
				Log:         `"$CLAUDE_PROJECT_DIR/.claude/hook-execution.log"`,
				ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
			},
		},
		{
			name:      "global is absolute and defers to the project hook",
			loc:       Location{GlobalDir: "/home/user/.claude"},
			separator: ':',
			want: hookset.ScriptValues{
				Bin:         `'/home/user/.claude/bin/pulsarules_cli'`,
				SkillsDirs:  `"$CLAUDE_PROJECT_DIR/.claude/skills":'/home/user/.claude/skills'`,
				Log:         `'/home/user/.claude/hook-execution.log'`,
				DeferTo:     `"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh"`,
				ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
			},
		},
		{
			name:      "global on windows quotes the separator",
			loc:       Location{GlobalDir: "C:/Users/user/.claude"},
			separator: ';',
			want: hookset.ScriptValues{
				Bin:         `'C:/Users/user/.claude/bin/pulsarules_cli'`,
				SkillsDirs:  `"$CLAUDE_PROJECT_DIR/.claude/skills"";"'C:/Users/user/.claude/skills'`,
				Log:         `'C:/Users/user/.claude/hook-execution.log'`,
				DeferTo:     `"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh"`,
				ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.loc.scriptValues(testCase.separator); got != testCase.want {
				t.Errorf("scriptValues = %+v, want %+v", got, testCase.want)
			}
		})
	}
}
