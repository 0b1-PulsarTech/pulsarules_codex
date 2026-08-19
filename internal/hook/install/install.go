package install

import (
	"fmt"
	"io/fs"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/report"
)

// Context carries the resolved inputs a hook Installer's Install needs,
// mirroring target.Context in internal/skill/target.
type Context struct {
	Dir          string
	Templates    fs.FS
	SettingsFile string
	GitHooks     []string
	// TypographicSeverity is baked into the generated hook scripts, so the gate
	// the hook applies is the one install chose (see githook.Options).
	TypographicSeverity string
	// BranchExtraTypes is baked into the pre-push hook the same way.
	BranchExtraTypes string
}

// UninstallContext carries the resolved inputs a hook Installer's Uninstall
// needs, the reversal counterpart of Context.
type UninstallContext struct {
	Dir          string
	SettingsFile string
}

// Installer is the consumer-declared port every hook target implements. It
// installs the hook infrastructure for one target type into ctx.Dir.
type Installer interface {
	// Name is the target identifier (e.g. "claude", "opencode", "git").
	Name() string
	// Install writes hooks, binaries, and config into ctx.Dir, returning a
	// Report of what it did - including any foreign file it backed up
	// rather than overwrote.
	Install(ctx Context) (report.Report, error)
	// Uninstall removes what Install wrote from ctx.Dir, reversing Install,
	// returning a Report of what it did. It is idempotent: a dir Install
	// never touched, or a second run, is a no-op with an empty Report.
	Uninstall(ctx UninstallContext) (report.Report, error)
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

// Install dispatches to the named installer, returning its Report. It
// errors when the name is not registered.
func (r *Registry) Install(name string, ctx Context) (report.Report, error) {
	inst, ok := r.byName[name]
	if !ok {
		return report.Report{}, fmt.Errorf("unknown hook installer %q", name)
	}
	rpt, err := inst.Install(ctx)
	if err != nil {
		return rpt, fmt.Errorf("install %s: %w", inst.Name(), err)
	}
	return rpt, nil
}

// Uninstall dispatches to the named installer's Uninstall, returning its
// Report. It errors when the name is not registered.
func (r *Registry) Uninstall(name string, ctx UninstallContext) (report.Report, error) {
	inst, ok := r.byName[name]
	if !ok {
		return report.Report{}, fmt.Errorf("unknown hook installer %q", name)
	}
	rpt, err := inst.Uninstall(ctx)
	if err != nil {
		return rpt, fmt.Errorf("uninstall %s: %w", inst.Name(), err)
	}
	return rpt, nil
}

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
