package hookwire

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

// hookScript is the basename used both to install the hook and to recognise (and
// replace) the hook's own entries during an idempotent settings merge.
const hookScript = "skill-router-reminder.sh"

// WireSettings merges the rendered hook entries into <claudeDir>/<settingsFile>
// ("settings.json" project-scope, "settings.local.json" per-machine). It is
// idempotent: any prior entry whose command references the hook script is
// dropped before fresh entries are appended, so re-running never duplicates.
// Existing permissions, enabledMcpjsonServers, and unrelated hooks are kept.
func WireSettings(templates fs.FS, claudeDir, settingsFile string) error {
	block, err := renderBlock(templates)
	if err != nil {
		return fmt.Errorf("render block: %w", err)
	}
	if err = os.MkdirAll(claudeDir, fsperm.DirPrivate); err != nil {
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
	if err = fsx.Save(path, merged); err != nil {
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

// mergeSettings folds the new hook block into the existing settings JSON,
// preserving every other key and hook event. Existing is empty for a fresh file.
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

// withoutHookScript drops the hook script's own commands from every group.
// why: a host/user may append another command inside our group, so this
// filters at the COMMAND level (like unwireGroups) instead of dropping the
// whole group and losing that neighbour. Unlike removal, an emptied group
// here IS ours to drop, since the fresh group replaces it.
func withoutHookScript(groups []hookGroup) []hookGroup {
	kept := make([]hookGroup, 0, len(groups))
	for _, group := range groups {
		cmds, removed := withoutHookCommand(group.Hooks)
		if !removed {
			kept = append(kept, group)
			continue
		}
		if len(cmds) == 0 {
			continue
		}
		group.Hooks = cmds
		kept = append(kept, group)
	}
	return kept
}
