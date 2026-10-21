package discovery

import (
	"fmt"
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type Lister interface {
	Root() string
	ListFiles() ([]string, error)
}

//nolint:ireturn // factory constructor returns interface
func Listed(
	repo Lister, skips, gitSkips source.SkipSet, logger *slog.Logger,
) (source.Provider, source.SkipSet) {
	root := repo.Root()
	paths, err := repo.ListFiles()
	if err != nil {
		logger.Warn(
			"walking the filesystem instead of git's file list",
			slog.Any("error", fmt.Errorf("list repository files: %w", err)),
		)
		return source.NewProvider(root, skips), skips
	}
	base := source.NewProvider(root, gitSkips)
	return source.NewListedProvider(base, paths, gitSkips), gitSkips
}
