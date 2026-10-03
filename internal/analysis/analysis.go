package analysis

import (
	"errors"
	"fmt"
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

func (s *Session) Analyze(scope Scope, status *vcs.Status, files FileSet) Result {
	d := s.Discover(scope, status, files)
	ctx := s.Load(d)

	sr := NewStageRunner(s.cfgAnalysis)
	sr.registerForScope(s.index, s.repo, scope)
	findings := sr.RunStages(ctx)

	if s.cfgAnalysis != nil && s.cfgAnalysis.IncludeGenerated {
		return Result{Findings: findings}
	}
	return splitGenerated(ctx, findings)
}

func (s *Session) Discover(scope Scope, status *vcs.Status, files FileSet) *Discovery {
	d := &Discovery{GitHistory: s.gitHistory(), Skips: s.skips}
	if s.repo == nil {
		return d
	}

	d.StagedRenames = s.stagedRenames()

	if files == FileSetAll {
		s.attachSources(d)
		d.ChangedFiles = walkAllFiles(d.Sources)
		return d
	}

	d.ChangedFiles = s.changedFiles(scope, status)
	// Commit scope only analyzes changed files; whole-tree analyzers on other scopes need Sources.
	if len(d.ChangedFiles) > 0 || scope != ScopeCommit {
		s.attachSources(d)
	}
	return d
}

func (s *Session) changedFiles(scope Scope, status *vcs.Status) []core.FileChange {
	if status == nil {
		fetched, err := s.repo.WorktreeStatus()
		if err != nil {
			s.logger.Warn("worktree status unavailable", slog.Any("error", err))
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

func (s *Session) Load(d *Discovery) *core.AnalysisContext {
	var root string
	if s.repo != nil {
		root = s.repo.Root()
	}

	ctx := &core.AnalysisContext{
		ProjectDir:    root,
		CommitMsg:     s.commitMsg,
		Config:        s.cfgAnalysis,
		Sources:       d.Sources,
		ChangedFiles:  d.ChangedFiles,
		GitHistory:    d.GitHistory,
		StagedRenames: d.StagedRenames,
		Skips:         d.Skips,
	}

	if len(d.ChangedFiles) > 0 && d.Sources != nil {
		markGenerated(ctx.ChangedFiles, d.Sources)
		ctx.ASTCache = populateASTCache(d.ChangedFiles, d.Sources)
	}
	return ctx
}

func (s *Session) attachSources(d *Discovery) {
	d.Sources, d.Skips = ListedSources(
		s.repo, s.skips, s.gitSkips, s.logger)
}

type fileLister interface {
	ListFiles() ([]string, error)
}

//nolint:ireturn // factory constructor returns interface
func ListedSources(
	repo vcs.Repository, skips, gitSkips sources.SkipSet, logger *slog.Logger,
) (sources.Provider, sources.SkipSet) {
	root := repo.Root()
	paths, err := listFiles(repo)
	if err != nil {
		logger.Warn("walking the filesystem instead of git's file list", slog.Any("error", err))
		return sources.NewProvider(root, skips), skips
	}
	base := sources.NewProvider(root, gitSkips)
	return sources.NewListedProvider(base, paths, gitSkips), gitSkips
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
