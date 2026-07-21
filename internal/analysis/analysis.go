package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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
	}
}

type Discovery struct {
	Sources       core.SourceProvider
	ChangedFiles  []core.FileChange
	GitHistory    []core.GitCommitEntry
	StagedRenames []core.Rename
}

func (s *Session) Analyze(scope Scope, status *vcs.Status, files FileSet) []core.Finding {
	d := s.Discover(scope, status, files)
	ctx := s.Load(d)

	sr := NewStageRunner(s.cfgAnalysis)
	sr.registerForScope(s.index, s.repo, scope)
	return sr.RunStages(ctx)
}

func (s *Session) Discover(scope Scope, status *vcs.Status, files FileSet) *Discovery {
	d := &Discovery{GitHistory: s.gitHistory()}
	if s.repo == nil {
		return d
	}

	const renameProbeScore = 1
	if renames, err := s.repo.StagedRenames(renameProbeScore); err == nil {
		d.StagedRenames = toCoreRenames(renames)
	}

	if scope == ScopeFull || scope == ScopeChanged {
		d.Sources = core.NewSourceProvider(s.repo.Root())
	}

	if files == FileSetAll && d.Sources != nil {
		d.ChangedFiles = walkAllFiles(d.Sources)
		return d
	}

	if status == nil {
		fetched, err := s.repo.WorktreeStatus()
		if err != nil {
			return d
		}
		status = &fetched
	}
	d.ChangedFiles = changesToFileChanges(status.Changes)
	return d
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
	}

	if len(d.ChangedFiles) > 0 && d.Sources != nil {
		ctx.ASTCache = populateASTCache(d.ChangedFiles, d.Sources)
	}
	return ctx
}
