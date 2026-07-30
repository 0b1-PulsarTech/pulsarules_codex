package host

import (
	"fmt"
	"slices"
)

type Registry struct {
	byName map[string]Target
}

func NewRegistry() *Registry {
	reg := &Registry{byName: map[string]Target{}}
	for _, tgt := range []Target{claudeTarget{}, opencodeTarget{}} {
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
