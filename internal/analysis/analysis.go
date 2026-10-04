package analysis

import (
	"log/slog"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type FileSet int

const (
	FileSetChanged FileSet = iota
	FileSetAll
)

type Session struct {
	repo        vcs.Repository
	commitMsg   string
	index       *knowledge.Index
	cfgAnalysis *core.AnalysisConfig
	skips       sources.SkipSet
	gitSkips    sources.SkipSet
	logger      *slog.Logger
}

func NewSession(
	repo vcs.Repository,
	commitMsg string,
	index *knowledge.Index,
	cfg *config.GovernanceConfig,
) *Session {
	if cfg == nil {
		cfg = config.Defaults()
		cfg.ApplyPreset()
	}
	return &Session{
		repo:        repo,
		commitMsg:   commitMsg,
		index:       index,
		cfgAnalysis: toAnalysisConfig(cfg),
		skips:       sources.NewSkipSet(sources.DefaultSkipDirs()),
		gitSkips:    sources.NewSkipSet(sources.GitSkipDirs()),
		logger:      slog.Default(),
	}
}

type Discovery struct {
	Sources       sources.Provider
	ChangedFiles  []core.FileChange
	GitHistory    []core.GitCommitEntry
	StagedRenames []core.Rename
	Skips         sources.SkipSet
}

func (sess *Session) Analyze(scope Scope, status *vcs.Status, files FileSet) Result {
	discovery := sess.Discover(scope, status, files)
	ctx := sess.Load(discovery)

	sr := NewStageRunner(sess.cfgAnalysis)
	sr.registerForScope(sess.index, sess.repo, scope)
	findings := sr.RunStages(ctx)

	if sess.cfgAnalysis != nil && sess.cfgAnalysis.IncludeGenerated {
		return Result{Findings: findings}
	}
	return splitGenerated(ctx, findings)
}

func (sess *Session) Discover(scope Scope, status *vcs.Status, files FileSet) *Discovery {
	discovery := &Discovery{GitHistory: sess.gitHistory(), Skips: sess.skips}
	if sess.repo == nil {
		return discovery
	}

	discovery.StagedRenames = sess.stagedRenames()

	if files == FileSetAll {
		sess.attachSources(discovery)
		discovery.ChangedFiles = walkAllFiles(discovery.Sources)
		return discovery
	}

	discovery.ChangedFiles = sess.changedFiles(scope, status)
	// Commit scope only analyzes changed files; whole-tree analyzers on other scopes need Sources.
	if len(discovery.ChangedFiles) > 0 || scope != ScopeCommit {
		sess.attachSources(discovery)
	}
	return discovery
}

func (sess *Session) changedFiles(scope Scope, status *vcs.Status) []core.FileChange {
	if status == nil {
		fetched, err := sess.repo.WorktreeStatus()
		if err != nil {
			sess.logger.Warn("worktree status unavailable", slog.Any("error", err))
			return nil
		}
		status = &fetched
	}
	changes := status.Changes
	if scope == ScopeCommit {
		changes = slices.DeleteFunc(slices.Clone(changes), func(change vcs.Change) bool {
			return !change.Staged
		})
	}
	return changesToFileChanges(changes)
}

func (sess *Session) Load(discovery *Discovery) *core.AnalysisContext {
	var root string
	if sess.repo != nil {
		root = sess.repo.Root()
	}

	ctx := &core.AnalysisContext{
		ProjectDir:    root,
		CommitMsg:     sess.commitMsg,
		Config:        sess.cfgAnalysis,
		Sources:       discovery.Sources,
		ChangedFiles:  discovery.ChangedFiles,
		GitHistory:    discovery.GitHistory,
		StagedRenames: discovery.StagedRenames,
		Skips:         discovery.Skips,
	}

	if len(discovery.ChangedFiles) > 0 && discovery.Sources != nil {
		markGenerated(ctx.ChangedFiles, discovery.Sources)
		ctx.ASTCache = populateASTCache(discovery.ChangedFiles, discovery.Sources)
	}
	return ctx
}
