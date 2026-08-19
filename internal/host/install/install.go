package install

import (
	"fmt"
	"io/fs"
	"slices"
)

type Context struct {
	Dir          string
	Templates    fs.FS
	SettingsFile string
	GitHooks     []string
	Warn         func(format string, args ...any)
}

type UninstallContext struct {
	Dir          string
	SettingsFile string
}

type Result struct {
	Removed         []string
	Restored        []string
	Notes           []string
	SettingsChanged bool
}

type Installer interface {
	Name() string
	Install(ctx Context) error
	Uninstall(ctx UninstallContext) (Result, error)
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

func (reg *Registry) Install(name string, ctx Context) error {
	inst, ok := reg.byName[name]
	if !ok {
		return fmt.Errorf("unknown hook installer %q", name)
	}
	if err := inst.Install(ctx); err != nil {
		return fmt.Errorf("install %s: %w", inst.Name(), err)
	}
	return nil
}

func (reg *Registry) Uninstall(name string, ctx UninstallContext) (Result, error) {
	inst, ok := reg.byName[name]
	if !ok {
		return Result{}, fmt.Errorf("unknown hook installer %q", name)
	}
	result, err := inst.Uninstall(ctx)
	if err != nil {
		return result, fmt.Errorf("uninstall %s: %w", inst.Name(), err)
	}
	return result, nil
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
