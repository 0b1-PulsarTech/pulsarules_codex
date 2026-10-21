package claude

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func UnwireSettings(claudeDir, settingsFile string) (bool, error) {
	path := filepath.Join(claudeDir, settingsFile)
	doc, err := jsondoc.LoadForRemoval(path)
	if err != nil {
		return false, err
	}
	if !doc.Exists() {
		return false, nil
	}

	changed, err := jsondoc.StripMap(doc, "hooks", unwireHooks)
	if err != nil {
		return false, fmt.Errorf("strip hooks: %w", err)
	}
	if !changed {
		return false, nil
	}
	if err = doc.SaveOrRemove(); err != nil {
		return false, fmt.Errorf("write settings: %w", err)
	}
	return true, nil
}

func unwireHooks(hooks map[string][]hookGroup) bool {
	changed := false
	for event, groups := range hooks {
		kept, eventChanged := dropHookScript(groups)
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

func dropHookScript(groups []hookGroup) ([]hookGroup, bool) {
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
		if strings.Contains(cmd.Command, reminderScriptAsset) {
			removed = true
			continue
		}
		kept = append(kept, cmd)
	}
	return kept, removed
}
