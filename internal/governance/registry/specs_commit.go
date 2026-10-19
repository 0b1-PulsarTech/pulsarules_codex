package registry

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/message"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func commitSpecs() []analyzerSpec {
	return []analyzerSpec{
		{
			id: "commit-lint",
			build: func(
				_ *knowledge.Index, _ vcs.Repository,
			) pass.Analyzer {
				catalog, err := emoji.NewCatalog()
				if err != nil {
					// simplification: analyzerBuilder returns only core.Analyzer; log
					// once and skip registering rather than widen that contract.
					// Upgrade path: give analyzerBuilder an error return.
					slog.Error("build commit emoji analyzer", slog.String("error", err.Error()))
					return nil
				}
				return message.NewAnalyzer(catalog)
			},
			gate: GateStatic,
		},
		{
			id: "commit-move-purity",
			build: func(
				_ *knowledge.Index, repo vcs.Repository,
			) pass.Analyzer {
				return movepurity.NewAnalyzer(repo)
			},
			gate: GateStatic,
		},
	}
}
