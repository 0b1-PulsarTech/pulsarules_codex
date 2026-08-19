package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// FileSet selects how Discover populates ChangedFiles: from git's worktree
// status (the fast path the Stop hook and pre-commit hook rely on) or by
// walking every source file in the project tree (a whole-repo governance
// run, which the tooling never gets from git status on a clean tree).
type FileSet int

const (
	// FileSetChanged populates ChangedFiles from git's worktree status.
	FileSetChanged FileSet = iota
	// FileSetAll populates ChangedFiles by walking the full source tree via
	// Sources, regardless of git status.
	FileSetAll
)

// Session orchestrates the analysis lifecycle: discovery, context loading,
// and analysis. It replaces ad-hoc entry points with a single path so all
// callers go through the same code.
type Session struct {
	repo        vcs.Repository
	commitMsg   string
	index       *knowledge.Index
	cfg         *config.GovernanceConfig
	cfgAnalysis *core.AnalysisConfig
}

// NewSession creates a session for repo. repo may be nil when no git
// repository is available (e.g. vcs.Open returned vcs.ErrNoRepository); the
// session then runs with no git history and no changed-file discovery
// rather than failing. When cfg is nil, default config is used with the
// recommended preset applied.
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
		cfg:         cfg,
		cfgAnalysis: toAnalysisConfig(cfg),
	}
}

// Discovery holds intermediate project state gathered before building the
// full AnalysisContext. Separating discovery from loading allows callers
// to inspect or modify the discovered state before analysis.
type Discovery struct {
	Sources       core.SourceProvider
	ChangedFiles  []core.FileChange
	GitHistory    []core.GitCommitEntry
	StagedRenames []core.Rename
}

// renameProbeScore is the similarity threshold passed to StagedRenames: low
// enough that a partial rename (e.g. 75% similar) still surfaces, so the
// move-purity analyzer can judge it against its own configured minimum
// instead of having git's detector silently drop it beforehand.
const renameProbeScore = 1

// Analyze runs the analyzers for scope and returns the findings a caller
// should act on, plus the count suppressed for generated files. status is
// the already-computed worktree status; pass nil to let Session read it
// itself. files selects git-status vs. a full source-tree walk.
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

// Discover gathers project-state information (changed files, git history,
// staged renames) and returns a Discovery. This phase does no AST parsing.
func (s *Session) Discover(scope Scope, status *vcs.Status, files FileSet) *Discovery {
	d := &Discovery{GitHistory: s.gitHistory()}
	if s.repo == nil {
		return d
	}

	// Every remaining scope reads file content: the static/AST analyzers are
	// registered for ScopeCommit too (staticScopes), so the pre-commit hook
	// needs Sources or they silently no-op on the very changeset they gate.
	if scope != ScopeFull && scope != ScopeChanged && scope != ScopeCommit {
		return d
	}

	if renames, err := s.repo.StagedRenames(renameProbeScore); err == nil {
		d.StagedRenames = toCoreRenames(renames)
	}

	d.Sources = core.NewSourceProvider(s.repo.Root())

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

// Load builds a full AnalysisContext from a Discovery, including AST cache
// population. Callers may modify the Discovery between Discover and Load.
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
		markGenerated(ctx.ChangedFiles, d.Sources)
		ctx.ASTCache = populateASTCache(d.ChangedFiles, d.Sources)
	}

	return ctx
}
