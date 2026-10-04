package governance

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func (runner *StageRunner) registerForScope(
	index *knowledge.Index,
	repo vcs.Repository,
	scope Scope,
) {
	langs := pass.NewLanguageRegistry()
	language.RegisterAll(langs)

	for _, analyzer := range registry.Analyzers(index, langs, repo, maxGateFor(scope)) {
		runner.Register(analyzer)
	}
}

func maxGateFor(scope Scope) []registry.Gate {
	switch scope {
	case ScopeCommit:
		// The pre-commit and commit-msg hooks run it, so only the fast static gate may.
		return []registry.Gate{registry.GateStatic}
	case ScopeChanged:
		// golangci-lint needs the whole project; branch-name guards a push.
		return []registry.Gate{registry.GateStatic, registry.GateArch}
	default:
		return []registry.Gate{registry.GateStatic, registry.GateArch, registry.GateFullOnly}
	}
}

func (sess *Session) attachSources(discovery *Request) {
	discovery.Sources, discovery.Skips = ListedSources(
		sess.repo, sess.skips, sess.gitSkips, sess.logger)
}

type fileLister interface {
	ListFiles() ([]string, error)
}

//nolint:ireturn // factory constructor returns interface
func ListedSources(
	repo vcs.Repository, skips, gitSkips source.SkipSet, logger *slog.Logger,
) (source.Provider, source.SkipSet) {
	root := repo.Root()
	paths, err := listFiles(repo)
	if err != nil {
		logger.Warn("walking the filesystem instead of git's file list", slog.Any("error", err))
		return source.NewProvider(root, skips), skips
	}
	base := source.NewProvider(root, gitSkips)
	return source.NewListedProvider(base, paths, gitSkips), gitSkips
}

func listFiles(repo vcs.Repository) ([]string, error) {
	lister, ok := repo.(fileLister)
	if !ok {
		return nil, fmt.Errorf("repository cannot list files: %w", errors.ErrUnsupported)
	}
	paths, err := lister.ListFiles()
	if err != nil {
		return nil, fmt.Errorf("list repository files: %w", err)
	}
	return paths, nil
}
