package claude

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func WireSettings(claudeDir, settingsFile string, loc Location) error {
	if err := os.MkdirAll(claudeDir, disk.ModeDirPrivate); err != nil {
		return fmt.Errorf("mkdir %q: %w", claudeDir, err)
	}
	path := filepath.Join(claudeDir, settingsFile)
	doc, err := jsondoc.Load(path)
	if err != nil {
		return jsondoc.PrefixParse(err, "merge settings: parse existing settings")
	}
	if err = mergeSettings(doc, buildHooksBlock(loc)); err != nil {
		return fmt.Errorf("merge settings: %w", err)
	}
	if err = doc.Save(); err != nil {
		return fmt.Errorf("save %q: %w", path, err)
	}
	return nil
}

func mergeSettings(doc *jsondoc.Document, block hooksBlock) error {
	hooks, err := jsondoc.Section[map[string][]hookGroup](doc, "hooks")
	if err != nil {
		return jsondoc.PrefixParse(err, "parse existing hooks")
	}
	if hooks == nil {
		hooks = map[string][]hookGroup{}
	}
	// For each event the block touches, drop any prior groups that reference the
	// hook script, then append the freshly-rendered groups.
	for event, groups := range block.Hooks {
		kept, _ := dropHookScript(hooks[event])
		hooks[event] = append(kept, groups...)
	}
	return jsondoc.SetSection(doc, "hooks", hooks)
}
