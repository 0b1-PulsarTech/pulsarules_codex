package registry

import (
	"slices"

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

type analyzerBuilder func(
	index *knowledge.Index,
	repo vcs.Repository,
) pass.Analyzer

type analyzerSpec struct {
	id       string
	build    analyzerBuilder
	gate     Gate
	internal bool
}

func Analyzers(
	index *knowledge.Index,
	repo vcs.Repository,
	gates []Gate,
) []pass.Analyzer {
	table := specs()
	built := make([]pass.Analyzer, 0, len(table))
	for _, spec := range table {
		if !slices.Contains(gates, spec.gate) {
			continue
		}
		if analyzer := spec.build(index, repo); analyzer != nil {
			built = append(built, analyzer)
		}
	}
	return built
}

func IDs() []string {
	table := specs()
	ids := make([]string, len(table))
	for idx, spec := range table {
		// A builder can return nil, so the id is declared rather than derived via ID().
		ids[idx] = spec.id
	}
	return ids
}

func ConfigurableIDs() []string {
	var ids []string
	for _, spec := range specs() {
		if !spec.internal {
			ids = append(ids, spec.id)
		}
	}
	return ids
}

func specs() []analyzerSpec {
	// Table order is run order; package-level-var dedupes against single-use-declaration.
	return slices.Concat(coreSpecs(), commitSpecs(), astSpecs(), styleSpecs(), pipelineSpecs())
}

func gatedSpec[Built pass.Analyzer](id string, gate Gate, ctor func() Built) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(_ *knowledge.Index, _ vcs.Repository) pass.Analyzer {
			return ctor()
		},
		gate: gate,
	}
}

func staticSpec[Built pass.Analyzer](id string, ctor func() Built) analyzerSpec {
	return gatedSpec(id, GateStatic, ctor)
}
