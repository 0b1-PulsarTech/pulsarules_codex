package claude

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type hookCommand struct {
	Type    string          `json:"type"`
	Command string          `json:"command"`
	Timeout json.RawMessage `json:"timeout,omitempty"`
}

type hookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

type hooksBlock struct {
	Hooks map[string][]hookGroup `json:"hooks"`
}

const StopTimeoutSeconds = 90

type hookRegistration struct {
	Event   string
	Matcher string
	Mode    string
	Timeout int
}

func hookRegistrations() []hookRegistration {
	return []hookRegistration{
		{Event: "SessionStart", Mode: "session-start"},
		{Event: "PreToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "pre-edit"},
		{Event: "PreToolUse", Matcher: "Grep|Glob|Bash", Mode: "pre-search"},
		{Event: "PostToolUse", Matcher: "Write|Edit|MultiEdit", Mode: "post-edit"},
		{Event: "UserPromptSubmit", Mode: "user-prompt"},
		{Event: "SubagentStart", Mode: "subagent-start"},
		// The host default is 600s and it discards output past the timeout; the Stop
		// deadline in internal/hook must stay under this by the margin the test pins.
		{Event: "Stop", Mode: "stop", Timeout: StopTimeoutSeconds},
		{Event: "SessionEnd", Mode: "session-end"},
	}
}

func RenderHooksBlock() ([]byte, error) {
	block := hooksBlock{Hooks: map[string][]hookGroup{}}
	for _, row := range hookRegistrations() {
		command := hookCommand{
			Type:    "command",
			Command: `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/` + hookScript + `" ` + row.Mode,
		}
		if row.Timeout > 0 {
			command.Timeout = json.RawMessage(strconv.Itoa(row.Timeout))
		}
		block.Hooks[row.Event] = append(block.Hooks[row.Event],
			hookGroup{Matcher: row.Matcher, Hooks: []hookCommand{command}})
	}
	out, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hooks block: %w", err)
	}
	return append(out, '\n'), nil
}
