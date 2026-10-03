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

func (analyzer *Analyzer) ID() string          { return "complexity" }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	th := analyzer.resolveThresholds(ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		return th.checkFile(ctx.ASTCache.FileSet(), fc, file)
	})
}

func (analyzer *Analyzer) resolveThresholds(ctx *pass.Context) thresholds {
	params := ctx.Params(analyzer.ID())
	return thresholds{
		maxComplexity: params.Int("max_complexity", analyzer.maxComplexity),
		maxFuncLines:  params.Int("max_func_lines", analyzer.maxFuncLines),
		maxParams:     params.Int("max_params", analyzer.maxParams),
		warnReporter:  pass.ResolveReporter(analyzer.warnReporter, ctx),
		infoReporter:  pass.ResolveReporter(analyzer.infoReporter, ctx),
	}
}
