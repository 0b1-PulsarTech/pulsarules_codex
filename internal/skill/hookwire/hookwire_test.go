package hookwire

import (
	"testing/fstest"
)

// fakeTemplates is the minimal templates filesystem the hookwire tests render
// against: the hook script, its README, and the settings block template.
func fakeTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/skill-router-reminder.sh": {Data: []byte("#!/usr/bin/env bash\nexit 0\n")},
		"hooks/README.md":                {Data: []byte("# why\n")},
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
