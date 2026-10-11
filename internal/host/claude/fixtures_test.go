package claude

import (
	"testing/fstest"
)

func routerHook(arg string) string {
	const command = `bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" `
	return `{"type": "command", "command": "` + command + arg + `"}`
}

func fakeTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/skill-router-reminder.sh.tmpl": {
			Data: []byte("#!/usr/bin/env bash\n# Installed by pulsarules_cli\n" +
				"bin=\"$CLAUDE_PROJECT_DIR/{{.BinaryRelPath}}\"\n" +
				"skills=\"$CLAUDE_PROJECT_DIR/{{.SkillsRelPath}}\"\nexit 0\n"),
		},
		"hooks/README.md": {
			Data: []byte("<!-- Installed by pulsarules_cli -->\n# why\n"),
		},
	}
}
