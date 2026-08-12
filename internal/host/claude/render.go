package claude

import (
	"encoding/json"
	"fmt"
	"io/fs"
)

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

type hooksBlock struct {
	Hooks map[string][]hookGroup `json:"hooks"`
}

func RenderHooksBlock(templates fs.FS) ([]byte, error) {
	tmpl, err := fs.ReadFile(templates, "hooks/settings.hooks.json.tmpl")
	if err != nil {
		return nil, fmt.Errorf("read hooks settings template: %w", err)
	}
	var block hooksBlock
	if err = json.Unmarshal(tmpl, &block); err != nil {
		return nil, fmt.Errorf("render hooks block: %w", err)
	}
	out, err := json.MarshalIndent(block, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hooks block: %w", err)
	}
	return append(out, '\n'), nil
}
