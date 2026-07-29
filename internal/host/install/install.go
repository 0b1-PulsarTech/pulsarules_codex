package install

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/claude"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/opencode"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/githook"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

type Installer interface {
	Name() string
	Install(dir string, templates fs.FS, settingsFile string) error
}

type Registry struct {
	byName map[string]Installer
}

func NewRegistry() *Registry {
	reg := &Registry{byName: map[string]Installer{}}
	for _, inst := range []Installer{claudeInstaller{}, opencodeInstaller{}, gitInstaller{}} {
		reg.byName[inst.Name()] = inst
	}
	return reg
}

func (reg *Registry) Install(name, dir string, templates fs.FS, settingsFile string) error {
	inst, ok := reg.byName[name]
	if !ok {
		return fmt.Errorf("unknown hook installer %q", name)
	}
	if err := inst.Install(dir, templates, settingsFile); err != nil {
		return fmt.Errorf("install %s: %w", inst.Name(), err)
	}
	return nil
}

func (reg *Registry) Has(name string) bool {
	_, ok := reg.byName[name]
	return ok
}

func (reg *Registry) Names() []string {
	names := make([]string, 0, len(reg.byName))
	for name := range reg.byName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

type claudeInstaller struct{}

func (claudeInstaller) Name() string { return "claude" }

func (claudeInstaller) Install(dir string, templates fs.FS, settingsFile string) error {
	if settingsFile == "" {
		settingsFile = "settings.json"
	}
	if err := claude.InstallHook(templates, dir); err != nil {
		return fmt.Errorf("install claude hook: %w", err)
	}
	if err := claude.WireSettings(templates, dir, settingsFile); err != nil {
		return fmt.Errorf("wire claude settings: %w", err)
	}
	if err := gitignore.Ensure(dir, "/bin/", "/hooks/"); err != nil {
		return fmt.Errorf("ensure claude gitignore: %w", err)
	}
	return nil
}

type opencodeInstaller struct{}

func (opencodeInstaller) Name() string { return "opencode" }

func (opencodeInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := opencode.Install(dir); err != nil {
		return fmt.Errorf("install opencode hook: %w", err)
	}
	return nil
}

type gitInstaller struct{}

func (gitInstaller) Name() string { return "git" }

func (gitInstaller) Install(dir string, _ fs.FS, _ string) error {
	if err := githook.Install(dir, []string{"commit-msg", "pre-commit"}); err != nil {
		return fmt.Errorf("install git hooks: %w", err)
	}
	return nil
}
