package complexity

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer reports cyclomatic complexity, function size, parameter
// count, flag arguments, and magic numbers in Go functions.
type Analyzer struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
}

const (
	defaultMaxComplexity = 15
	defaultMaxFuncLines  = 80
	defaultMaxParams     = 5
)

var (
	complexityWarnReporter = core.NewReporter("complexity", core.SeverityWarning, core.CatAST)
	complexityInfoReporter = core.NewReporter("complexity", core.SeverityInfo, core.CatAST)
)

// NewAnalyzer creates a complexity analyzer with default thresholds.
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		maxComplexity: defaultMaxComplexity,
		maxFuncLines:  defaultMaxFuncLines,
		maxParams:     defaultMaxParams,
	}
}

func (a *Analyzer) ID() string   { return "complexity" }
func (a *Analyzer) Name() string { return "Code complexity" }
func (a *Analyzer) Description() string {
	return "Reports cyclomatic complexity, long functions, many parameters, flag arguments"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageAST }
func (a *Analyzer) Category() core.Category { return core.CatAST }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	th := a.resolveThresholds(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return th.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

// resolveThresholds overlays the runtime analyzer params onto the compiled-in
// defaults so a project can tighten or loosen the thresholds without a
// rebuild.
func (a *Analyzer) resolveThresholds(ctx *core.AnalysisContext) thresholds {
	params := ctx.Params(a.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", a.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", a.maxFuncLines),
		maxParams:     params.Int("max_params", a.maxParams),
	}
}
