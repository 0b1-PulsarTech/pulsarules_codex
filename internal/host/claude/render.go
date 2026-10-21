package claude

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
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

func buildHooksBlock() hooksBlock {
	block := hooksBlock{Hooks: map[string][]hookGroup{}}
	for _, row := range hook.Events() {
		command := hookCommand{
			Type:    "command",
			Command: `bash "$CLAUDE_PROJECT_DIR/.claude/hooks/` + reminderScriptAsset + `" ` + row.Mode,
		}
		if row.Timeout > 0 {
			command.Timeout = json.RawMessage(strconv.Itoa(row.Timeout))
		}
		block.Hooks[row.Host] = append(block.Hooks[row.Host],
			hookGroup{Matcher: row.Matcher, Hooks: []hookCommand{command}})
	}
	return block
}

func RenderHooksBlock() ([]byte, error) {
	out, err := json.MarshalIndent(buildHooksBlock(), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hooks block: %w", err)
	}
	return append(out, '\n'), nil
}
