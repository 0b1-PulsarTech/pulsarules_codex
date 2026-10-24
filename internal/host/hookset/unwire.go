package hookset

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func Unwire(dir, settingsFile string) (bool, error) {
	path := filepath.Join(dir, settingsFile)
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

func unwireHooks(hooks map[string][]Group) bool {
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

func dropHookScript(groups []Group) ([]Group, bool) {
	kept := make([]Group, 0, len(groups))
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

func withoutHookCommand(cmds []Command) ([]Command, bool) {
	kept := make([]Command, 0, len(cmds))
	removed := false
	for _, cmd := range cmds {
		if strings.Contains(cmd.Command, ScriptName) {
			removed = true
			continue
		}
		kept = append(kept, cmd)
	}
	return kept, removed
}
