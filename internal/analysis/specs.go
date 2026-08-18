package analysis

import (
	"log/slog"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/complexity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/controlflow"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/imports"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/namedreturns"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/naming"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/shadowing"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/timediscipline"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/bigcomment"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/filesize"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/noemdash"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/simplificationpath"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/static/topoffile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// analyzerSpecs is the full ordered list of every registered analyzer and the
// scopes it runs under: this file's text/AST-check table followed by
// pipelineAnalyzerSpecs (specs_pipeline.go), split into two files purely to
// keep each under the file-size ceiling - the append below is still the one
// place a new analyzer gets registered, not a second duplicated list.
//
// why: this stays a plain table rather than injector bindings resolved via
// remy.GetAll with tags. The table already solves the problem that move
// would target - two duplicated per-scope registration lists where a missed
// line silently dropped an analyzer - so moving it into a tag-keyed graph
// would add indirection without a correctness gain. Deliberate, not an
// oversight.
var analyzerSpecs = append([]analyzerSpec{
	{
		id: "file-size",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return filesize.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "no-em-dash",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return noemdash.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "import-groups",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return imports.NewAnalyzer("github.com/0b1-PulsarTech/pulsarules_codex")
		},
		scopes: staticScopes,
	},
	{
		id: "naming",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return naming.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "top-of-file",
		build: func(_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return topoffile.NewAnalyzer(langs)
		},
		scopes: staticScopes,
	},
	{
		id: "big-comment",
		build: func(_ *knowledge.Index, langs *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return bigcomment.NewAnalyzer(langs)
		},
		scopes: staticScopes,
	},
	{
		id: "simplification-path",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return simplificationpath.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "commit-lint",
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
		id: "commit-move-purity",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, repo vcs.Repository) core.Analyzer {
			return movepurity.NewAnalyzer(repo)
		},
		scopes: staticScopes,
	},
	{
		id: "control-flow",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return controlflow.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "shadowing",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return shadowing.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "complexity",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return complexity.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "named-returns",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return namedreturns.NewAnalyzer()
		},
		scopes: staticScopes,
	},
	{
		id: "time-discipline",
		build: func(_ *knowledge.Index, _ *core.LanguageRegistry, _ vcs.Repository) core.Analyzer {
			return timediscipline.NewAnalyzer()
		},
		scopes: staticScopes,
	},
}, pipelineAnalyzerSpecs...)
