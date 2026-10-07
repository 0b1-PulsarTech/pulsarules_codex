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

func (analyzer *Analyzer) ID() string          { return "file-size" }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	limits := newLineLimits(analyzer.resolveMaxLines(ctx))
	reporter := analyzer.reporter.Resolved(ctx)
	eligible := core.ExtensionFilter(ctx, analyzer.ID(), core.DefaultSourceExtensions())
	return core.EachChangedFile(ctx, eligible, func(
		fc core.FileChange, src []byte,
	) []core.Finding {
		limit := limits.forFile(fc)
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

func (analyzer *Analyzer) resolveMaxLines(ctx *core.AnalysisContext) int {
	return ctx.Params(analyzer.ID()).Int("max_lines", analyzer.maxLines)
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

func (limits lineLimits) forFile(fc core.FileChange) int {
	if fc.IsTest {
		return limits.test
	}
	return limits.production
}

func countLines(src []byte) int {
	return bytes.Count(src, []byte{'\n'})
}
