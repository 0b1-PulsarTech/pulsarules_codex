package codex

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
)

func TestLocation_ScriptValues(t *testing.T) {
	t.Parallel()

	const codexDir = `$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)`
	const root = `$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)`
	testCases := []struct {
		name      string
		loc       Location
		separator rune
		want      hookset.ScriptValues
	}{
		{
			name:      "project resolves everything from the script's own location",
			separator: ':',
			want: hookset.ScriptValues{
				Bin:         `"` + codexDir + `/bin/pulsarules_cli"`,
				SkillsDirs:  `"` + root + `/.agents/skills":"$HOME/.agents/skills"`,
				Log:         `"` + codexDir + `/hook-execution.log"`,
				ProjectRoot: `"` + root + `"`,
			},
		},
		{
			name:      "global is absolute and follows the working directory",
			loc:       Location{GlobalDir: "/h/.codex", UserSkills: "/h/.agents/skills"},
			separator: ':',
			want: hookset.ScriptValues{
				Bin:         `'/h/.codex/bin/pulsarules_cli'`,
				SkillsDirs:  `"$PWD/.agents/skills":'/h/.agents/skills'`,
				Log:         `'/h/.codex/hook-execution.log'`,
				ProjectRoot: `"$PWD"`,
			},
		},
		{
			name:      "a semicolon list separator is quoted",
			loc:       Location{GlobalDir: "/h/.codex", UserSkills: "/h/s"},
			separator: ';',
			want: hookset.ScriptValues{
				Bin:         `'/h/.codex/bin/pulsarules_cli'`,
				SkillsDirs:  `"$PWD/.agents/skills"";"'/h/s'`,
				Log:         `'/h/.codex/hook-execution.log'`,
				ProjectRoot: `"$PWD"`,
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

func TestLocation_Command(t *testing.T) {
	t.Parallel()

	const walk = `d=$PWD; case $d in /*) ;; *) exit 0;; esac; while [ "$d" != / ] && ` +
		`[ ! -f "$d/.codex/hooks/skill-router-reminder.sh" ]; do d=$(dirname "$d"); done; `
	testCases := []struct {
		name string
		loc  Location
		want string
	}{
		{
			name: "project walks up and stays quiet outside a checkout",
			want: walk + `[ -f "$d/.codex/hooks/skill-router-reminder.sh" ] || exit 0; ` +
				`PULSARULES_HOOK_DIALECT=codex bash "$d/.codex/hooks/skill-router-reminder.sh" stop`,
		},
		{
			name: "global yields to a runnable project install and quotes its own path",
			loc:  Location{GlobalDir: "/h/o'neil/.codex"},
			want: walk + `if [ -f "$d/.codex/hooks/skill-router-reminder.sh" ] && ` +
				`[ -x "$d/.codex/bin/pulsarules_cli" ] && ` +
				`[ ! "$d/.codex/hooks/skill-router-reminder.sh" -ef ` +
				`'/h/o'\''neil/.codex/hooks/skill-router-reminder.sh' ]; then exit 0; fi; ` +
				`[ -f '/h/o'\''neil/.codex/hooks/skill-router-reminder.sh' ] || exit 0; ` +
				`PULSARULES_HOOK_DIALECT=codex bash ` +
				`'/h/o'\''neil/.codex/hooks/skill-router-reminder.sh' stop`,
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

func TestLocation_BlockMatchesGolden(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		loc    Location
		golden string
	}{
		{"project", Location{}, "testdata/project_hooks.golden.json"},
		{
			"global",
			Location{GlobalDir: "/home/user/.codex", UserSkills: "/home/user/.agents/skills"},
			"testdata/global_hooks.golden.json",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			want, err := os.ReadFile(testCase.golden)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			got, err := RenderHooksBlock(testCase.loc)
			if err != nil {
				t.Fatalf("RenderHooksBlock: %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("hooks block differs from the golden:\n%s", got)
			}
		})
	}
}

func TestLocation_BlockSpeaksCodex(t *testing.T) {
	t.Parallel()

	raw, err := RenderHooksBlock(Location{})
	if err != nil {
		t.Fatalf("RenderHooksBlock: %v", err)
	}
	var block hookset.Block
	if err = json.Unmarshal(raw, &block); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(block.Hooks) != len(hook.Events())-1 {
		t.Errorf("events = %d, want one per distinct host event", len(block.Hooks))
	}
	for event, groups := range block.Hooks {
		for _, group := range groups {
			assertCodexGroup(t, event, group)
		}
	}
}

func assertCodexGroup(tb testing.TB, event string, group hookset.Group) {
	tb.Helper()
	command := group.Hooks[0]
	if !strings.Contains(command.Command, "PULSARULES_HOOK_DIALECT=codex") {
		tb.Errorf("%s command lacks the codex dialect: %q", event, command.Command)
	}
	if strings.Contains(command.Command, "CLAUDE_PROJECT_DIR") {
		tb.Errorf("%s command leans on a Claude variable: %q", event, command.Command)
	}
	wantTimeout := map[string]string{
		"SessionEnd": "3", "Stop": "90", "PreToolUse": "10", "PostToolUse": "10",
		"UserPromptSubmit": "10",
	}[event]
	if string(command.Timeout) != wantTimeout {
		tb.Errorf("%s timeout = %q, want %q", event, command.Timeout, wantTimeout)
	}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		event hook.Event
		want  string
	}{
		{"no matcher stays none", hook.Event{Mode: "stop"}, ""},
		{
			"edit events match the patch tool",
			hook.Event{Matcher: "Write|Edit", Mode: "pre-edit"},
			"^(apply_patch|Edit|Write|MultiEdit)$",
		},
		{
			"post edit too",
			hook.Event{Matcher: "Write|Edit", Mode: "post-edit"},
			"^(apply_patch|Edit|Write|MultiEdit)$",
		},
		{"search is the shell tool", hook.Event{Matcher: "Grep|Bash", Mode: "pre-search"}, "Bash"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := matcher(testCase.event); got != testCase.want {
				t.Errorf("matcher = %q, want %q", got, testCase.want)
			}
		})
	}
}
