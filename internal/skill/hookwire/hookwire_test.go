package hookwire

import (
	"testing/fstest"
)

// fakeTemplates is the minimal templates filesystem the hookwire tests render
// against: the hook script, its README, and the settings block template.
// The script fixture carries the same {{.BinaryRelPath}} / {{.SkillsRelPath}}
// placeholders as the real embedded template, so InstallHook exercises the
// same render path here as in production.
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
		"hooks/settings.hooks.json.tmpl": {Data: []byte(`{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" session-start"}]}
    ],
    "PreToolUse": [
      {"matcher": "Write|Edit|MultiEdit", "hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" pre-edit"}]}
    ],
    "PostToolUse": [
      {"matcher": "Write|Edit|MultiEdit", "hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" post-edit"}]}
    ],
    "UserPromptSubmit": [
      {"hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" user-prompt"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/skill-router-reminder.sh\" stop"}]}
    ]
  }
}
`)},
	}
}
