package governance

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
)

func (sess *Session) stagedRenames() []pass.Rename {
	const renameProbeScore = 1
	// Low so a partial rename still reaches move-purity, which judges it against its own
	// minimum instead of git's detector dropping it first.
	renames, err := sess.repo.StagedRenames(renameProbeScore)
	if err != nil {
		sess.logger.Warn("staged renames unavailable", slog.Any("error", err))
		return nil
	}
	return discovery.ToCoreRenames(renames)
}
