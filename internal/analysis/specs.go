package analysis

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/arch"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ruleinjection"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// analyzerSpecs is the single ordered list of every registered analyzer and
// the scopes it runs under. Adding an analyzer is a one-line addition here.
//
// why: this stays a plain table rather than injector bindings resolved via
// remy.GetAll with tags. The table already solves the problem that move
// would target - two duplicated per-scope registration lists where a missed
// line silently dropped an analyzer - so moving it into a tag-keyed graph
// would add indirection without a correctness gain. Deliberate, not an
// oversight.
var analyzerSpecs = []analyzerSpec{
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return filesize.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return noemdash.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return imports.NewAnalyzer("github.com/0b1-PulsarTech/pulsarules_codex")
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return naming.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return topoffile.NewAnalyzer(langs)
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return bigcomment.NewAnalyzer(langs)
		},
		scopes: staticScopes,
	},
	{
		// simplification: analyzerBuilder returns only core.Analyzer, with no
		// error and no injected logger, shared across every spec in this table -
		// so a catalog load failure cannot propagate through registerForScope /
		// Session.Analyze without widening that contract for every other
		// analyzer. Log the failure once here (the only point that observes it)
		// and skip registering the analyzer, rather than falling back to an
		// empty catalog, which would reject every commit. Upgrade path: give
		// analyzerBuilder an error return if a load failure ever needs to fail
		// the whole run loudly instead of just disabling this one check.
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			catalog, err := emoji.NewCatalog()
			if err != nil {
				slog.Error("build commit emoji analyzer", slog.String("error", err.Error()))
				return nil
			}
			return commit.NewAnalyzer(catalog)
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, repo vcs.Repository) core.Analyzer {
			return movepurity.NewAnalyzer(repo)
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return controlflow.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return shadowing.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return complexity.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return namedreturns.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return arch.NewPackageBoundaryAnalyzer()
		},
		scopes: archScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return arch.NewImportCycleAnalyzer()
		},
		scopes: archScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return delegation.NewGolangcilintAnalyzer("golangci-lint")
		},
		scopes: delegationScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return delegation.NewGoplsAnalyzer()
		},
		scopes: delegationScopes,
	},
	{
		build: func(index *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return ruleinjection.NewAnalyzer(index)
		},
		scopes: staticScopes,
	},
	{
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return output.NewAnalyzer()
		},
		scopes: staticScopes,
	},
}
