package complexity

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
	warnReporter  core.Reporter
	infoReporter  core.Reporter
}

const (
	defaultMaxComplexity = 15
	defaultMaxFuncLines  = 80
	defaultMaxParams     = 5
)

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		maxComplexity: defaultMaxComplexity,
		maxFuncLines:  defaultMaxFuncLines,
		maxParams:     defaultMaxParams,
		warnReporter:  core.NewReporter("complexity", core.SeverityWarning, core.CatAST),
		infoReporter:  core.NewReporter("complexity", core.SeverityInfo, core.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "complexity" }
func (a *Analyzer) Stage() core.StageID { return core.StageAST }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	th := a.resolveThresholds(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return th.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func (a *Analyzer) resolveThresholds(ctx *core.AnalysisContext) thresholds {
	params := ctx.Params(a.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", a.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", a.maxFuncLines),
		maxParams:     params.Int("max_params", a.maxParams),
		warnReporter:  a.warnReporter,
		infoReporter:  a.infoReporter,
	}
}
