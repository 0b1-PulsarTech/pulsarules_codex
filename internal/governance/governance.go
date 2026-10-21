package governance

import (
	"log/slog"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/generated"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/registry"
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

type Request struct {
	Scope     Scope
	Status    *vcs.Status
	Files     FileSet
	CommitMsg string
}

type Session struct {
	repo        Repository
	index       *knowledge.Index
	catalog     *emoji.Catalog
	cfgAnalysis *pass.AnalysisConfig
	skips       source.SkipSet
	gitSkips    source.SkipSet
	logger      *slog.Logger
	providers   providerFactory
	policy      testfile.Classifier
	seed        []finding.Finding
}

func NewSession(
	repo Repository,
	index *knowledge.Index,
	catalog *emoji.Catalog,
	cfg *config.GovernanceConfig,
) *Session {
	if cfg == nil {
		cfg = config.Defaults()
	}
	sess := &Session{
		repo:        repo,
		index:       index,
		catalog:     catalog,
		cfgAnalysis: cfg.Analysis(),
		skips:       source.NewSkipSet(append(source.DefaultSkipDirs(), cfg.SkipDirs...)),
		gitSkips:    source.NewSkipSet(append(source.GitSkipDirs(), cfg.SkipDirs...)),
		logger:      slog.Default(),
		providers:   defaultProviders,
	}
	policy, err := testfile.New(sess.cfgAnalysis.TestGlobs)
	if err != nil {
		sess.seed = []finding.Finding{configFinding(err)}
		policy = testfile.Defaults()
	}
	sess.policy = policy
	return sess
}

func (sess *Session) Analyze(req Request) findings.Report {
	ctx := sess.load(req)
	detector := generated.NewDetector(ctx.Sources)
	detector.Mark(ctx.ChangedFiles)

	deps := registry.Deps{Index: sess.index, Repo: sess.repo, Emoji: sess.catalog}
	pipe := newPipeline(
		sess.cfgAnalysis, registry.Analyzers(deps, maxGateFor(req.Scope)), sess.transforms(),
	)
	reported := pipe.run(ctx, slices.Clone(sess.seed))

	if sess.cfgAnalysis.IncludeGenerated {
		return findings.Report{Findings: reported}
	}
	kept, suppressed := detector.Split(reported)
	return findings.Report{Findings: kept, Suppressed: suppressed}
}

func (sess *Session) load(req Request) *pass.Context {
	ctx := &pass.Context{
		CommitMsg:  req.CommitMsg,
		Config:     sess.cfgAnalysis,
		GitHistory: sess.gitHistory(req.CommitMsg),
		Skips:      sess.skips,
		TestPolicy: sess.policy,
	}
	if sess.repo == nil {
		return ctx
	}
	ctx.ProjectDir = sess.repo.Root()
	ctx.StagedRenames = sess.stagedRenames()
	sess.discover(ctx, req)

	analyzable := slices.DeleteFunc(
		slices.Clone(ctx.ChangedFiles),
		func(file pass.FileChange) bool { return ctx.Skips.SkipsPath(file.Path) },
	)
	if len(analyzable) > 0 && ctx.Sources != nil {
		ctx.ASTCache = populateASTCache(analyzable, ctx.Sources)
		ctx.Syntax = syntax.NewCache(
			sess.providers(ctx.ASTCache),
			syntax.Corpus{
				Root:    ctx.ProjectDir,
				Sources: changedSources(analyzable),
				Read:    ctx.Sources.Read,
				Module:  func() string { return ctx.GoModule().Path },
			},
		)
	}
	return ctx
}

func (sess *Session) discover(ctx *pass.Context, req Request) {
	if req.Files == FileSetAll {
		sess.attachSources(ctx)
		ctx.ChangedFiles = discovery.WalkAllFiles(ctx.Sources, sess.policy)
		return
	}

	ctx.ChangedFiles = sess.changedFiles(req)
	// Commit scope only analyzes changed files; whole-tree analyzers on other scopes need Sources.
	if len(ctx.ChangedFiles) > 0 || req.Scope != ScopeCommit {
		sess.attachSources(ctx)
	}
}

func (sess *Session) changedFiles(req Request) []pass.FileChange {
	status := req.Status
	if status == nil {
		fetched, err := sess.repo.WorktreeStatus()
		if err != nil {
			sess.logger.Warn("worktree status unavailable", slog.Any("error", err))
			return nil
		}
		status = &fetched
	}
	changes := status.Changes
	if req.Scope == ScopeCommit {
		changes = slices.DeleteFunc(slices.Clone(changes), func(change vcs.Change) bool {
			return !change.Staged
		})
	}
	return discovery.ChangesToFileChanges(changes, sess.policy)
}

func configFinding(err error) finding.Finding {
	return config.Finding(projectfile.Name, err.Error(), "fix the test globs")
}
