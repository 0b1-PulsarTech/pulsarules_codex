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

// analyzerSpecs is the full ordered list of every registered analyzer,
// split across this file and specs_pipeline.go for file-size only.
//
// why: a plain table, not remy.GetAll tags - avoids the missed-line bug a
// duplicated per-scope list caused; tags add indirection, no gain.
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
			return imports.NewAnalyzer()
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
		// simplification: analyzerBuilder returns only core.Analyzer (no error,
		// no logger), so a load failure here can't widen that contract for
		// every analyzer. Log it once and skip registering, instead of falling
		// back to an empty catalog that would reject every commit. Upgrade
		// path: give analyzerBuilder an error return if this ever needs to fail loudly.
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
