package hookset

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook"
)

const (
	HooksSubdir = "hooks"
	BinSubdir   = "bin"
)

type Command struct {
	Type    string          `json:"type"`
	Command string          `json:"command"`
	Timeout json.RawMessage `json:"timeout,omitempty"`
}

type Group struct {
	Matcher string    `json:"matcher,omitempty"`
	Hooks   []Command `json:"hooks"`
}

type Block struct {
	Hooks map[string][]Group `json:"hooks"`
}

func BuildBlock(command func(mode string) string, matcher func(hook.Event) string) Block {
	block := Block{Hooks: map[string][]Group{}}
	for _, row := range hook.Events() {
		entry := Command{Type: "command", Command: command(row.Mode)}
		if row.Timeout > 0 {
			entry.Timeout = json.RawMessage(strconv.Itoa(row.Timeout))
		}
		block.Hooks[row.Host] = append(block.Hooks[row.Host],
			Group{Matcher: matcher(row), Hooks: []Command{entry}})
	}
	return block
}

func RenderBlock(block Block) ([]byte, error) {
	out, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hooks block: %w", err)
	}
	return append(out, '\n'), nil
}
