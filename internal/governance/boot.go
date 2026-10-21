package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/dartsyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func maxGateFor(scope Scope) registry.Gate {
	switch scope {
	case ScopeCommit:
		// The pre-commit and commit-msg hooks run it, so only the fast static gate may.
		return registry.GateStatic
	case ScopeChanged:
		// golangci-lint and delegated commands need the whole project; branch-name guards a push.
		return registry.GateArch
	default:
		return registry.GateFullOnly
	}
}

type providerFactory func(*astcache.Cache) []syntax.Provider

func defaultProviders(cache *astcache.Cache) []syntax.Provider {
	return []syntax.Provider{gosyntax.New(cache), dartsyntax.New(), tssyntax.New()}
}

func (sess *Session) attachSources(found *Request) {
	found.Sources, found.Skips = discovery.Listed(
		sess.repo, sess.skips, sess.gitSkips, sess.logger)
}

type Repository interface {
	registry.Repository
	discovery.Lister
	WorktreeStatus() (vcs.Status, error)
	StagedRenames(minScore int) ([]vcs.Rename, error)
	RecentSubjects(limit int) ([]string, error)
	HeadAuthorEpoch() (int64, bool, error)
}
