package governance

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func populateASTCache(changedFiles []pass.FileChange, provider source.Provider) *astcache.Cache {
	if len(changedFiles) == 0 {
		return nil
	}
	cache := astcache.New()
	// A _test.go file is parsed too: time-discipline polices clocks in tests.
	for _, fc := range changedFiles {
		if fc.Extension != ".go" {
			continue
		}
		src, ok := provider.Read(fc.Path)
		if !ok {
			continue
		}
		if _, err := cache.Parse(fc.Path, src); err != nil {
			continue
		}
	}
	return cache
}

func changedSources(files []pass.FileChange) []syntax.Source {
	sources := make([]syntax.Source, len(files))
	for idx, file := range files {
		sources[idx] = syntax.Source{Path: file.Path, IsTest: file.IsTest}
	}
	return sources
}

func (sess *Session) stagedRenames() []pass.Rename {
	const renameProbeScore = 1
	// Low so a partial rename still reaches move-purity, which judges it against its own
	// minimum instead of git's detector dropping it first.
	renames, err := sess.repo.StagedRenames(renameProbeScore)
	if err != nil {
		sess.logger.Warn("staged renames unavailable", slog.Any("error", err))
		return nil
	}
	return discovery.FromRenames(renames)
}
