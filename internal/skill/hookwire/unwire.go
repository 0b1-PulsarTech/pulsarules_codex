package hookwire

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
)

func UnwireSettings(claudeDir, settingsFile string) (bool, error) {
	path := filepath.Join(claudeDir, settingsFile)
	existing, err := os.ReadFile(path) //nolint:gosec // path is under the caller's .claude dir.
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read %q: %w", path, err)
	}

	settings := map[string]json.RawMessage{}
	if err = json.Unmarshal(existing, &settings); err != nil {
		return false, fmt.Errorf("%w: %q: %w", fsx.ErrUnparseableJSON, path, err)
	}

	changed, err := fsx.StripMapSection(
		settings,
		"hooks",
		fmt.Sprintf("%q hooks", path),
		unwireHooks,
	)
	if err != nil {
		return false, fmt.Errorf("strip hooks: %w", err)
	}
	if !changed {
		return false, nil
	}
	if _, err = fsx.SaveOrRemove(path, settings); err != nil {
		return false, fmt.Errorf("write settings: %w", err)
	}
	return true, nil
}

func unwireHooks(hooks map[string][]hookGroup) bool {
	changed := false
	for event, groups := range hooks {
		kept, eventChanged := unwireGroups(groups)
		if !eventChanged {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = kept
		}
	}
	return changed
}

func unwireGroups(groups []hookGroup) ([]hookGroup, bool) {
	kept := make([]hookGroup, 0, len(groups))
	changed := false
	for _, group := range groups {
		cmds, removed := withoutHookCommand(group.Hooks)
		// An untouched group is kept even when empty; dropping it would delete the file.
		if !removed {
			kept = append(kept, group)
			continue
		}
		changed = true
		if len(cmds) == 0 {
			continue
		}
		group.Hooks = cmds
		kept = append(kept, group)
	}
	return kept, changed
}

func withoutHookCommand(cmds []hookCommand) ([]hookCommand, bool) {
	kept := make([]hookCommand, 0, len(cmds))
	removed := false
	for _, cmd := range cmds {
		if strings.Contains(cmd.Command, hookScript) {
			removed = true
			continue
		}
		kept = append(kept, cmd)
	}
	return kept, removed
}
