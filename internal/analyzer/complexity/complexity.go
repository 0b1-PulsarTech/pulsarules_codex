package complexity

import (
	"go/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	maxComplexity int
	maxFuncLines  int
	maxParams     int
	warnReporter  pass.Reporter
	infoReporter  pass.Reporter
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
		warnReporter:  pass.NewReporter("complexity", pass.SeverityWarning, pass.CatAST),
		infoReporter:  pass.NewReporter("complexity", pass.SeverityInfo, pass.CatAST),
	}
}

func (a *Analyzer) ID() string          { return "complexity" }
func (a *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	th := a.resolveThresholds(ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		return th.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func (a *Analyzer) resolveThresholds(ctx *pass.Context) thresholds {
	params := ctx.Params(a.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", a.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", a.maxFuncLines),
		maxParams:     params.Int("max_params", a.maxParams),
		warnReporter:  a.warnReporter,
		infoReporter:  a.infoReporter,
	}
}
