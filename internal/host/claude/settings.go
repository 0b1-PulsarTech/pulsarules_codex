package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

const hookScript = "skill-router-reminder.sh"

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

func WireSettings(templates fs.FS, claudeDir, settingsFile string) error {
	block, err := renderBlock(templates)
	if err != nil {
		return fmt.Errorf("render block: %w", err)
	}
	if err = os.MkdirAll(claudeDir, disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", claudeDir, err)
	}
	path := filepath.Join(claudeDir, settingsFile)
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's .claude dir.
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("read %q: %w", path, err)
	}

	merged, err := mergeSettings(existing, block)
	if err != nil {
		return fmt.Errorf("merge settings: %w", err)
	}
	if err = jsondoc.Save(path, merged); err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	return nil
}

func renderBlock(templates fs.FS) (hooksBlock, error) {
	raw, err := RenderHooksBlock(templates)
	if err != nil {
		return hooksBlock{}, err
	}
	var block hooksBlock
	if unmarshalErr := json.Unmarshal(raw, &block); unmarshalErr != nil {
		return hooksBlock{}, fmt.Errorf("decode rendered hooks block: %w", unmarshalErr)
	}
	return block, nil
}

func mergeSettings(existing []byte, block hooksBlock) (map[string]json.RawMessage, error) {
	settings := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(existing)) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return nil, fmt.Errorf("parse existing settings: %w", err)
		}
	}

	hooks := map[string][]hookGroup{}
	if raw, ok := settings["hooks"]; ok {
		if err := json.Unmarshal(raw, &hooks); err != nil {
			return nil, fmt.Errorf("parse existing hooks: %w", err)
		}
	}

	// For each event the block touches, drop any prior groups that reference the
	// hook script, then append the freshly-rendered groups.
	for event, groups := range block.Hooks {
		hooks[event] = append(withoutHookScript(hooks[event]), groups...)
	}

	rawHooks, err := json.Marshal(hooks)
	if err != nil {
		return nil, fmt.Errorf("marshal hooks: %w", err)
	}
	settings["hooks"] = rawHooks
	return settings, nil
}

func withoutHookScript(groups []hookGroup) []hookGroup {
	kept := make([]hookGroup, 0, len(groups))
	for _, group := range groups {
		if groupReferencesHook(group) {
			continue
		}
		kept = append(kept, group)
	}
	return kept
}

func groupReferencesHook(group hookGroup) bool {
	for _, cmd := range group.Hooks {
		if strings.Contains(cmd.Command, hookScript) {
			return true
		}
	}
	return false
}
