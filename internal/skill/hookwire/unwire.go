package hookwire

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsonconfig"
)

// UnwireSettings undoes WireSettings on <claudeDir>/<settingsFile>, reporting
// whether it changed the file and dropping an emptied group, event, "hooks"
// key or file in turn. Invalid JSON returns a wrapped fsx.ErrUnparseableJSON
// and changes nothing; an absent file is a silent no-op.
// why: filters at COMMAND level, but an emptied group survives unless we emptied it.
func UnwireSettings(claudeDir, settingsFile string) (bool, error) {
	path := filepath.Join(claudeDir, settingsFile)
	existing, err := jsonconfig.Read(path)
	if err != nil {
		return false, err
	}
	if existing == nil {
		return false, nil
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

// unwireHooks drops the hook script's command from every group in hooks, in
// place, dropping an emptied group and an emptied event along with it. It
// reports whether it changed anything.
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

// why: an untouched group is kept verbatim even when it holds no commands -
// "ended up empty" is not "we emptied it", and dropping it deletes the file.
func unwireGroups(groups []hookGroup) ([]hookGroup, bool) {
	kept := make([]hookGroup, 0, len(groups))
	changed := false
	for _, group := range groups {
		cmds, removed := withoutHookCommand(group.Hooks)
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

// withoutHookCommand drops every command referencing the hook script from
// cmds, reporting whether it dropped any - the command-level counterpart to
// groupReferencesHook's group-level check.
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
