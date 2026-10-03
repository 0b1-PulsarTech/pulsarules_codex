package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
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
	}
}

type Request struct {
	Sources       source.SourceProvider
	ChangedFiles  []pass.FileChange
	GitHistory    []pass.GitCommitEntry
	StagedRenames []pass.Rename
}

func (s *Session) Analyze(scope Scope, status *vcs.Status, files FileSet) Result {
	d := s.Discover(scope, status, files)
	ctx := s.Load(d)

	sr := newPipeline(s.cfgAnalysis)
	sr.registerForScope(s.index, s.repo, scope)
	findings := sr.RunStages(ctx)

	if s.cfgAnalysis != nil && s.cfgAnalysis.IncludeGenerated {
		return Result{Findings: findings}
	}
	return splitGenerated(ctx, findings)
}

func (s *Session) Discover(scope Scope, status *vcs.Status, files FileSet) *Request {
	d := &Request{GitHistory: s.gitHistory()}
	if s.repo == nil {
		return d
	}

	const renameProbeScore = 1
	if renames, err := s.repo.StagedRenames(renameProbeScore); err == nil {
		d.StagedRenames = discovery.ToCoreRenames(renames)
	}

	d.Sources = source.NewSourceProvider(s.repo.Root(), s.skips)

	if files == FileSetAll {
		d.ChangedFiles = discovery.WalkAllFiles(d.Sources)
		return d
	}

	if status == nil {
		fetched, err := s.repo.WorktreeStatus()
		if err != nil {
			return d
		}
		status = &fetched
	}
	d.ChangedFiles = discovery.ChangesToFileChanges(status.Changes)
	return d
}

func (s *Session) Load(d *Request) *pass.Context {
	var root string
	if s.repo != nil {
		root = s.repo.Root()
	}

	ctx := &pass.Context{
		ProjectDir:    root,
		CommitMsg:     s.commitMsg,
		Config:        s.cfgAnalysis,
		Sources:       d.Sources,
		ChangedFiles:  d.ChangedFiles,
		GitHistory:    d.GitHistory,
		StagedRenames: d.StagedRenames,
		Skips:         s.skips,
	}

	if len(d.ChangedFiles) > 0 && d.Sources != nil {
		discovery.MarkGenerated(ctx.ChangedFiles, d.Sources)
		ctx.ASTCache = populateASTCache(d.ChangedFiles, d.Sources)
	}
	return ctx
}
