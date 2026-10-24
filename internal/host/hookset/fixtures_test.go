package hookset

import (
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
)

func routerHook(arg string) string {
	const command = `bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" `
	return `{"type": "command", "command": "` + command + arg + `"}`
}

func testBlock() Block {
	return BuildBlock(
		func(mode string) string {
			return `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh" ` + mode
		},
		func(event hook.Event) string { return event.Matcher },
	)
}

func testValues() ScriptValues {
	return ScriptValues{
		Bin:         `"$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_cli"`,
		SkillsDirs:  `"$CLAUDE_PROJECT_DIR/.claude/skills"`,
		Log:         `"$CLAUDE_PROJECT_DIR/.claude/hook-execution.log"`,
		ProjectRoot: `"$CLAUDE_PROJECT_DIR"`,
	}
}

func fakeTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/skill-router-reminder.sh.tmpl": {
			Data: []byte("#!/usr/bin/env bash\n# Installed by pulsarules_cli\n" +
				"bin={{.Bin}}\n" +
				"skills={{.SkillsDirs}}\nexit 0\n"),
		},
		"hooks/README.md": {
			Data: []byte("<!-- Installed by pulsarules_cli -->\n# why\n"),
		},
	}
}
