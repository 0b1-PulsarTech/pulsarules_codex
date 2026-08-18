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

// UnwireSettings removes the hook script's entries from
// <claudeDir>/<settingsFile>, undoing WireSettings. Unlike the install-time
// merge (whose withoutHookScript drops an entire hookGroup when any of its
// commands reference the hook script - safe there only because the group is
// immediately re-appended), this filters at the COMMAND level: a group that
// also carries an unrelated command keeps that command, since removal never
// re-appends anything. It drops a group left with no commands, an event left
// with no groups, and the "hooks" key itself left with no events, mirroring
// how mergeSettings shapes the file, and deletes the file when nothing but
// our entries was ever in it. It wraps fsx.ErrUnparseableJSON and makes no
// change when the file is not valid JSON, and is a silent no-op when the file
// or its "hooks" key is absent, so re-running is never an error. It reports
// whether it actually changed the file, so a caller can tell a real removal
// from a no-op against a settings file the hook was never wired into.
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
