package install

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

type Context struct {
	Dir                 string
	Templates           fs.FS
	SettingsFile        string
	GitHooks            []string
	TypographicSeverity string
	BranchExtraTypes    string
}

type UninstallContext struct {
	Dir          string
	SettingsFile string
}

type Installer interface {
	Name() string
	Install(ctx Context) (report.Report, error)
	Uninstall(ctx UninstallContext) (report.Report, error)
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

func (reg *Registry) Install(name string, ctx Context) (report.Report, error) {
	inst, ok := reg.byName[name]
	if !ok {
		return report.Report{}, fmt.Errorf("unknown hook installer %q", name)
	}
	rpt, err := inst.Install(ctx)
	if err != nil {
		return rpt, fmt.Errorf("install %s: %w", inst.Name(), err)
	}
	return rpt, nil
}

func (reg *Registry) Uninstall(name string, ctx UninstallContext) (report.Report, error) {
	inst, ok := reg.byName[name]
	if !ok {
		return report.Report{}, fmt.Errorf("unknown hook installer %q", name)
	}
	rpt, err := inst.Uninstall(ctx)
	if err != nil {
		return rpt, fmt.Errorf("uninstall %s: %w", inst.Name(), err)
	}
	return rpt, nil
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
