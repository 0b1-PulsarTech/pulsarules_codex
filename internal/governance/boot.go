package governance

import (
	"fmt"
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/dartsyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (runner *StageRunner) registerForScope(
	index *knowledge.Index,
	repo Repository,
	scope Scope,
) {
	for _, analyzer := range registry.Analyzers(index, repo, maxGateFor(scope)) {
		runner.Register(analyzer)
	}
}

func maxGateFor(scope Scope) []registry.Gate {
	switch scope {
	case ScopeCommit:
		// The pre-commit and commit-msg hooks run it, so only the fast static gate may.
		return []registry.Gate{registry.GateStatic}
	case ScopeChanged:
		// golangci-lint and delegated commands need the whole project; branch-name guards a push.
		return []registry.Gate{registry.GateStatic, registry.GateArch}
	default:
		return []registry.Gate{registry.GateStatic, registry.GateArch, registry.GateFullOnly}
	}
}

type providerFactory func(*astcache.Cache) []syntax.Provider

func defaultProviders(cache *astcache.Cache) []syntax.Provider {
	return []syntax.Provider{gosyntax.New(cache), dartsyntax.New(), tssyntax.New()}
}

func (sess *Session) attachSources(discovery *Request) {
	discovery.Sources, discovery.Skips = ListedSources(
		sess.repo, sess.skips, sess.gitSkips, sess.logger)
}

type (
	Repository interface {
		registry.Repository
		Listing
		WorktreeStatus() (vcs.Status, error)
		StagedRenames(minScore int) ([]vcs.Rename, error)
		RecentSubjects(limit int) ([]string, error)
		HeadAuthorEpoch() (int64, bool, error)
	}
	Listing interface {
		Root() string
		ListFiles() ([]string, error)
	}
)

//nolint:ireturn // factory constructor returns interface
func ListedSources(
	repo Listing, skips, gitSkips source.SkipSet, logger *slog.Logger,
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
