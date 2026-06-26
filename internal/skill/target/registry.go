package target

import (
	"fmt"
	"slices"
)

// Registry maps a layout name to its install Strategy. Build it once at boot
// with NewRegistry and pass it where it is needed; there is no package-level
// registry var (no Singleton, no Service Locator).
type Registry struct {
	byName map[string]Target
}

// NewRegistry builds the Registry with the built-in Claude and opencode layouts,
// keying each Strategy by its own Name so the allow-list cannot drift.
func NewRegistry() *Registry {
	reg := &Registry{byName: map[string]Target{}}
	for _, tgt := range []Target{claudeTarget{}, opencodeTarget{}} {
		reg.byName[tgt.Name()] = tgt
	}
	return reg
}

// Install dispatches to the named layout's Strategy, returning its Report. It
// errors when the name is not registered.
func (r *Registry) Install(name string, ctx Context) (Report, error) {
	tgt, ok := r.byName[name]
	if !ok {
		return Report{}, fmt.Errorf("unknown target %q", name)
	}
	report, err := tgt.Install(ctx)
	if err != nil {
		return report, fmt.Errorf("install target %q: %w", name, err)
	}
	return report, nil
}

// Has reports whether a layout name is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.byName[name]
	return ok
}

// Names returns the registered layout names sorted for stable diagnostics.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.byName))
	for name := range r.byName {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
