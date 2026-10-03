package governance

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
	langs *pass.LanguageRegistry,
	repo vcs.Repository,
) pass.Analyzer

type analyzerSpec struct {
	id    string
	build analyzerBuilder
	gate  Gate
}

func Analyzers(
	index *knowledge.Index,
	langs *pass.LanguageRegistry,
	repo vcs.Repository,
	gates []Gate,
) []pass.Analyzer {
	table := specs()
	built := make([]pass.Analyzer, 0, len(table))
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
	return slices.Concat(coreSpecs(), commitSpecs(), astSpecs(), pipelineSpecs())
}

func gatedSpec[Built pass.Analyzer](id string, gate Gate, ctor func() Built) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(_ *knowledge.Index, _ *pass.LanguageRegistry, _ vcs.Repository) pass.Analyzer {
			return ctor()
		},
		gate: gate,
	}
}

func staticSpec[Built pass.Analyzer](id string, ctor func() Built) analyzerSpec {
	return gatedSpec(id, GateStatic, ctor)
}

func langSpec[Built pass.Analyzer](
	id string, ctor func(*pass.LanguageRegistry) Built,
) analyzerSpec {
	return analyzerSpec{
		id: id,
		build: func(
			_ *knowledge.Index, langs *pass.LanguageRegistry, _ vcs.Repository,
		) pass.Analyzer {
			return ctor(langs)
		},
		gate: GateStatic,
	}
}
