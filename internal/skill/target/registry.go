package target

import (
	"fmt"
	"slices"
)

type Registry struct {
	byName map[string]Target
}

func NewRegistry() *Registry {
	reg := &Registry{byName: map[string]Target{}}
	for _, tgt := range []Target{
		claudeTarget{}, opencodeTarget{}, agentsTarget{}, cursorTarget{},
	} {
		reg.byName[tgt.Name()] = tgt
	}
	return reg
}

func (reg *Registry) Install(name string, ctx Context) (Report, error) {
	tgt, ok := reg.byName[name]
	if !ok {
		return Report{}, fmt.Errorf("unknown target %q", name)
	}
	report, err := tgt.Install(ctx)
	if err != nil {
		return report, fmt.Errorf("install target %q: %w", name, err)
	}
	return report, nil
}

func (reg *Registry) Uninstall(name string, ctx UninstallContext) (Report, error) {
	tgt, ok := reg.byName[name]
	if !ok {
		return Report{}, fmt.Errorf("unknown target %q", name)
	}
	report, err := tgt.Uninstall(ctx)
	if err != nil {
		return report, fmt.Errorf("uninstall target %q: %w", name, err)
	}
	return report, nil
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

func (reg *Registry) DetectTargets(base string) []string {
	var found []string
	for _, name := range reg.Names() {
		if reg.byName[name].Present(base) {
			found = append(found, name)
		}
	}
	return found
}
