package registry

import (
	"errors"
	"fmt"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Gate int

const (
	GateStatic Gate = iota
	GateArch
	GateFullOnly
)

type Repository interface {
	CurrentBranch() (string, error)
	Blob(side vcs.Side, path string) (string, error)
	Entry(side vcs.Side, path string) (mode, id string, err error)
	Dir(side vcs.Side, dir string) ([]string, error)
}

type Deps struct {
	Index *knowledge.Index
	Repo  Repository
}

type entry struct {
	pass.Descriptor
	build    func(Deps) (pass.Analyzer, error)
	gate     Gate
	internal bool
}

type Built struct {
	ID       string
	Analyzer pass.Analyzer
}

func Analyzers(deps Deps, gates []Gate) ([]Built, error) {
	return build(specs(), deps, gates)
}

func build(table []entry, deps Deps, gates []Gate) ([]Built, error) {
	built := make([]Built, 0, len(table))
	var failures []error
	for _, spec := range table {
		if !slices.Contains(gates, spec.gate) {
			continue
		}
		analyzer, err := spec.build(deps)
		if err != nil {
			failures = append(failures, fmt.Errorf("build %s: %w", spec.ID, err))
			continue
		}
		built = append(built, Built{ID: spec.ID, Analyzer: analyzer})
	}
	return built, errors.Join(failures...)
}

func IDs() []string {
	table := specs()
	ids := make([]string, len(table))
	for idx, spec := range table {
		ids[idx] = spec.ID
	}
	return ids
}

func ConfigurableIDs() []string {
	var ids []string
	for _, spec := range specs() {
		if !spec.internal {
			ids = append(ids, spec.ID)
		}
	}
	return ids
}

func Params() map[string][]string {
	params := make(map[string][]string)
	for _, spec := range specs() {
		if len(spec.Params) > 0 {
			params[spec.ID] = slices.Clone(spec.Params)
		}
	}
	return params
}

func DelegatedFormats() []string {
	return command.Formats()
}

func plain[Built pass.Analyzer](
	desc pass.Descriptor, gate Gate, ctor func() Built,
) entry {
	return viaDeps(desc, gate, func(Deps) pass.Analyzer { return ctor() })
}

func viaDeps(desc pass.Descriptor, gate Gate, ctor func(Deps) pass.Analyzer) entry {
	return entry{
		Descriptor: desc,
		gate:       gate,
		build:      func(deps Deps) (pass.Analyzer, error) { return ctor(deps), nil },
	}
}

func hide(spec entry) entry {
	spec.internal = true
	return spec
}
