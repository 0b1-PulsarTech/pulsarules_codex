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
		"hooks/skill-router-reminder.sh": {Data: []byte("#!/usr/bin/env bash\nexit 0\n")},
		"hooks/README.md":                {Data: []byte("# why\n")},
		"hooks/settings.hooks.json.tmpl": {Data: []byte(`{
  "hooks": {
    "SessionStart": [
      {"hooks": [` + routerHook("session-start") + `]}
    ],
    "PreToolUse": [
      {"matcher": "Write|Edit|MultiEdit", "hooks": [` + routerHook("pre-edit") + `]}
    ],
    "PostToolUse": [
      {"matcher": "Write|Edit|MultiEdit", "hooks": [` + routerHook("post-edit") + `]}
    ],
    "UserPromptSubmit": [
      {"hooks": [` + routerHook("user-prompt") + `]}
    ],
    "Stop": [
      {"hooks": [` + routerHook("stop") + `]}
    ]
  }
}
`)},
	}
}
