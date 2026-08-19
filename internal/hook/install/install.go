package install

import (
	"fmt"
	"io/fs"
	"slices"
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
	// Warn, if set, receives a formatted non-fatal notice - such as a
	// foreign file backed up before being overwritten - so the caller's own
	// report channel surfaces it instead of the message being dropped. A nil
	// Warn silently discards the notice.
	Warn func(format string, args ...any)
}

// UninstallContext carries the resolved inputs a hook Installer's Uninstall
// needs, the reversal counterpart of Context.
type UninstallContext struct {
	Dir          string
	SettingsFile string
}

// Result reports what an Installer's Uninstall actually removed from disk,
// the reversal counterpart of what Install wrote. Removed is empty when
// there was nothing of this installer's to remove - a no-op against a
// directory Install never touched - so a caller can tell a real removal
// from a no-op instead of assuming success from a nil error.
type Result struct {
	Removed []string
	// Restored carries a ready-to-print message for every backup Uninstall
	// restored to its original path, undoing a prior Install-time backup
	// (see Context.Warn).
	Restored []string
	// Notes carries ready-to-print observations an uninstall could not act on
	// itself - a leftover an operator has to reconcile by hand.
	Notes []string
	// SettingsChanged reports whether Uninstall actually unwired the specific
	// settings file named by UninstallContext.SettingsFile, distinct from a
	// shared one-time cleanup Removed may also carry (claudeInstaller's
	// gitignore entries). A caller looping over multiple files must key its
	// report off this, not len(Removed). Only claudeInstaller sets it.
	SettingsChanged bool
}

// Installer is the consumer-declared port every hook target implements. It
// installs the hook infrastructure for one target type into ctx.Dir.
type Installer interface {
	// Name is the target identifier (e.g. "claude", "opencode", "git").
	Name() string
	// Install writes hooks, binaries, and config into ctx.Dir.
	Install(ctx Context) error
	// Uninstall removes what Install wrote from ctx.Dir, reversing Install. It
	// is idempotent: a dir Install never touched, or a second run, is a no-op.
	Uninstall(ctx UninstallContext) (Result, error)
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
func (r *Registry) Install(name string, ctx Context) error {
	inst, ok := r.byName[name]
	if !ok {
		return fmt.Errorf("unknown hook installer %q", name)
	}
	if err := inst.Install(ctx); err != nil {
		return fmt.Errorf("install %s: %w", inst.Name(), err)
	}
	return nil
}

// Uninstall dispatches to the named installer's Uninstall, returning its
// Result. It errors when the name is not registered.
func (r *Registry) Uninstall(name string, ctx UninstallContext) (Result, error) {
	inst, ok := r.byName[name]
	if !ok {
		return Result{}, fmt.Errorf("unknown hook installer %q", name)
	}
	result, err := inst.Uninstall(ctx)
	if err != nil {
		return result, fmt.Errorf("uninstall %s: %w", inst.Name(), err)
	}
	return result, nil
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
