package filesize

import (
	"bytes"
	"fmt"
	"math"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct {
	maxLines int
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxLines = 180
	return &Analyzer{
		maxLines: defaultMaxLines,
		reporter: core.NewReporter("file-size", core.SeverityWarning, core.CatProject),
	}
}

func (a *Analyzer) ID() string          { return "file-size" }
func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	limits := newLineLimits(a.resolveMaxLines(ctx))
	reporter := a.reporter.Resolved(ctx)
	eligible := core.ExtensionFilter(ctx, a.ID(), core.DefaultSourceExtensions())
	return core.EachChangedFile(ctx, eligible, func(
		fc core.FileChange, src []byte,
	) []core.Finding {
		limit := limits.forFile(fc.Path)
		lines := countLines(src)
		if lines <= limit {
			return nil
		}
		return []core.Finding{reporter.At(
			fc.Path,
			0,
			fmt.Sprintf("file is %d lines, max %d", lines, limit),
			fmt.Sprintf("split into smaller files (max %d lines per file)", limit),
		)}
	})
}

func (a *Analyzer) resolveMaxLines(ctx *core.AnalysisContext) int {
	return ctx.Params(a.ID()).Int("max_lines", a.maxLines)
}

type lineLimits struct {
	production int
	test       int
}

func newLineLimits(maxLines int) lineLimits {
	const testLineMultiplier = 2.6
	return lineLimits{
		production: maxLines,
		// A table-driven test carries a case table with no counterpart in the code under
		// test; one shared number would push the table out of the file it documents.
		test: int(math.Round(float64(maxLines) * testLineMultiplier)),
	}
}

func (l lineLimits) forFile(path string) int {
	if core.IsTestPath(path) {
		return l.test
	}
	return l.production
}

func countLines(src []byte) int {
	return bytes.Count(src, []byte{'\n'})
}
