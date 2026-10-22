package registry

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
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
	Files(side vcs.Side, dir string) ([]string, error)
}

type Deps struct {
	Index *knowledge.Index
	Repo  Repository
	Emoji *emoji.Catalog
}

type entry struct {
	pass.Descriptor
	build    func(Deps) pass.Analyzer
	gate     Gate
	internal bool
}

type Built struct {
	ID       string
	Analyzer pass.Analyzer
}

func Analyzers(deps Deps, maxGate Gate) []Built {
	return build(specs(), deps, maxGate)
}

func build(table []entry, deps Deps, maxGate Gate) []Built {
	built := make([]Built, 0, len(table))
	for _, spec := range table {
		// Gates are ordered: a scope runs every analyzer at or below its own gate.
		if spec.gate > maxGate {
			continue
		}
		built = append(built, Built{ID: spec.ID, Analyzer: spec.build(deps)})
	}
	return built
}

func specIDs() []string {
	table := specs()
	ids := make([]string, len(table))
	for idx, spec := range table {
		ids[idx] = spec.ID
	}
	return ids
}

func configurableIDs() []string {
	var ids []string
	for _, spec := range specs() {
		if !spec.internal {
			ids = append(ids, spec.ID)
		}
	}
	return ids
}

func specParams() map[string][]string {
	params := make(map[string][]string)
	for _, spec := range specs() {
		if len(spec.Params) > 0 {
			params[spec.ID] = slices.Clone(spec.Params)
		}
	}
	return params
}

func plain[Concrete pass.Analyzer](
	desc pass.Descriptor, gate Gate, ctor func() Concrete,
) entry {
	return viaDeps(desc, gate, func(Deps) pass.Analyzer { return ctor() })
}

func viaDeps(desc pass.Descriptor, gate Gate, ctor func(Deps) pass.Analyzer) entry {
	return entry{
		Descriptor: desc,
		gate:       gate,
		build:      ctor,
	}
}

func hide(spec entry) entry {
	spec.internal = true
	return spec
}
