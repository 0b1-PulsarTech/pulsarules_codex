package governance

import (
	"context"
	"log/slog"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/testfile"
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
	cfgAnalysis *pass.AnalysisConfig
	skips       source.SkipSet
	gitSkips    source.SkipSet
	logger      *slog.Logger
	providers   providerFactory
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
		skips:       source.NewSkipSet(source.DefaultSkipDirs()),
		gitSkips:    source.NewSkipSet(source.GitSkipDirs()),
		logger:      slog.Default(),
		providers:   defaultProviders,
	}
}

type Request struct {
	Sources       source.Provider
	ChangedFiles  []pass.FileChange
	GitHistory    []pass.GitCommitEntry
	StagedRenames []pass.Rename
	Skips         source.SkipSet
}

func (sess *Session) Analyze(scope Scope, status *vcs.Status, files FileSet) Result {
	discovery := sess.Discover(scope, status, files)
	ctx := sess.Load(discovery)

	sr := newPipeline(sess.cfgAnalysis)
	sr.registerForScope(sess.index, sess.repo, scope)
	findings := sr.RunStages(ctx)

	if sess.cfgAnalysis != nil && sess.cfgAnalysis.IncludeGenerated {
		return Result{Findings: findings}
	}
	return splitGenerated(ctx, findings)
}

func (sess *Session) Discover(scope Scope, status *vcs.Status, files FileSet) *Request {
	discovery := &Request{GitHistory: sess.gitHistory(), Skips: sess.skips}
	if sess.repo == nil {
		return discovery
	}

	discovery.StagedRenames = sess.stagedRenames()

	if files == FileSetAll {
		sess.attachSources(discovery)
		discovery.ChangedFiles = walkAllFiles(discovery.Sources, testfile.Defaults())
		return discovery
	}

	discovery.ChangedFiles = sess.changedFiles(scope, status)
	// Commit scope only analyzes changed files; whole-tree analyzers on other scopes need Sources.
	if len(discovery.ChangedFiles) > 0 || scope != ScopeCommit {
		sess.attachSources(discovery)
	}
	return discovery
}

func (sess *Session) changedFiles(scope Scope, status *vcs.Status) []pass.FileChange {
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
	return changesToFileChanges(changes, testfile.Defaults())
}

func (sess *Session) Load(discovery *Request) *pass.Context {
	var root string
	if sess.repo != nil {
		root = sess.repo.Root()
	}

	analyzable := slices.DeleteFunc(
		slices.Clone(discovery.ChangedFiles),
		func(file pass.FileChange) bool { return discovery.Skips.SkipsPath(file.Path) },
	)
	ctx := &pass.Context{
		ProjectDir:    root,
		CommitMsg:     sess.commitMsg,
		Config:        sess.cfgAnalysis,
		Sources:       discovery.Sources,
		ChangedFiles:  discovery.ChangedFiles,
		GitHistory:    discovery.GitHistory,
		StagedRenames: discovery.StagedRenames,
		Skips:         discovery.Skips,
		TestPolicy:    testfile.Defaults(),
	}

	if len(analyzable) > 0 && discovery.Sources != nil {
		markGenerated(ctx.ChangedFiles, discovery.Sources)
		ctx.ASTCache = populateASTCache(analyzable, discovery.Sources)
		ctx.Syntax = syntax.NewCache(
			context.Background(),
			sess.providers(ctx.ASTCache),
			syntax.Corpus{
				Root:    root,
				Sources: changedSources(analyzable),
				Read:    discovery.Sources.Read,
			},
		)
	}
	return ctx
}
