package install

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/githook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install/opencodehook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/hookwire"
)

// Installer is the consumer-declared port every hook target implements. It
// installs the hook infrastructure for one target type into the given project
// directory. templates is the embedded templates filesystem; settingsFile is
// the target-specific settings file name (empty when not applicable).
type Installer interface {
	// Name is the target identifier (e.g. "claude", "opencode", "git").
	Name() string
	// Install writes hooks, binaries, and config into dir.
	Install(dir string, templates fs.FS, settingsFile string) error
}

// Registry maps target names to their hook installers. Build it once at boot
// with NewRegistry and pass it where needed.
type Registry struct {
	byName map[string]Installer
}

// NewRegistry builds the Registry with the built-in Claude, opencode, and git
// hook installers, keying each by its own Name.
func NewRegistry() *Registry {
	reg := &Registry{byName: map[string]Installer{}}
	for _, inst := range []Installer{claudeInstaller{}, opencodeInstaller{}, gitInstaller{}} {
		reg.byName[inst.Name()] = inst
	}
	return reg
}

// Install dispatches to the named installer. It errors when the name is not
// registered.
func (r *Registry) Install(name, dir string, templates fs.FS, settingsFile string) error {
	inst, ok := r.byName[name]
	if !ok {
		return fmt.Errorf("unknown hook installer %q", name)
	}
	if err := inst.Install(dir, templates, settingsFile); err != nil {
		return fmt.Errorf("install %s: %w", inst.Name(), err)
	}
	return nil
}

// Has reports whether a hook installer is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.byName[name]
	return ok
}

// Names returns the registered installer names sorted for stable diagnostics.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// claudeInstaller installs the Claude Code hook: shell script + binary +
// settings.json wiring + .gitignore.
type claudeInstaller struct{}

func (claudeInstaller) Name() string { return "claude" }

func (claudeInstaller) Install(dir string, templates fs.FS, settingsFile string) error {
	claudeDir := dir
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	if err := hookwire.InstallHook(templates, claudeDir); err != nil {
		return fmt.Errorf("install claude hook: %w", err)
	}
	if err := hookwire.WireSettings(templates, claudeDir, settingsFile); err != nil {
		return fmt.Errorf("wire claude settings: %w", err)
	}
	if err := gitignore.Ensure(claudeDir, "/bin/", "/hooks/"); err != nil {
		return fmt.Errorf("ensure claude gitignore: %w", err)
	}
	return nil
}

// opencodeInstaller installs the opencode governance plugin + binary +
// .gitignore.
type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := opencodehook.Install(dir); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}

// gitInstaller installs git hook scripts into .git/hooks/.
type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

func (gitInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := githook.Install(dir, []string{"commit-msg", "pre-commit"}); err != nil {
		return fmt.Errorf("install git hooks: %w", err)
	}
	return nil
}
