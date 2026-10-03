package registry

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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
	langs *core.LanguageRegistry,
	repo vcs.Repository,
) core.Analyzer

type analyzerSpec struct {
	id    string
	build analyzerBuilder
	gate  Gate
}

func Analyzers(
	index *knowledge.Index,
	langs *core.LanguageRegistry,
	repo vcs.Repository,
	gates []Gate,
) []core.Analyzer {
	table := specs()
	built := make([]core.Analyzer, 0, len(table))
	for _, spec := range table {
		if !slices.Contains(gates, spec.gate) {
			continue
		}
		if analyzer := spec.build(index, langs, repo); analyzer != nil {
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

func specs() []analyzerSpec {
	// Table order is run order; package-level-var dedupes against single-use-declaration.
	return slices.Concat(coreSpecs(), commitSpecs(), astSpecs(), styleSpecs(), pipelineSpecs())
}

func gatedSpec[Built core.Analyzer](id string, gate Gate, ctor func() Built) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return ctor()
		},
		gate: gate,
	}
}

func staticSpec[Built core.Analyzer](id string, ctor func() Built) analyzerSpec {
	return gatedSpec(id, GateStatic, ctor)
}

func langSpec[Built core.Analyzer](
	id string, ctor func(*core.LanguageRegistry) Built,
) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(
			_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository,
		) core.Analyzer {
			return ctor(langs)
		},
		gate: GateStatic,
	}
}
