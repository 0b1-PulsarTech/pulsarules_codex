package filesize

import (
	"bytes"
	"fmt"
	"math"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	maxLines int
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxLines = 180
	return &Analyzer{
		maxLines: defaultMaxLines,
		reporter: finding.NewReporter("file-size", finding.SeverityWarning, finding.CatProject),
	}
}

func (analyzer *Analyzer) ID() string             { return "file-size" }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	limits := newLineLimits(analyzer.resolveMaxLines(ctx))
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	eligible := pass.ExtensionFilter(ctx, analyzer.ID(), pass.DefaultSourceExtensions())
	return pass.EachChangedFile(ctx, eligible, func(
		fc pass.FileChange, src []byte,
	) []finding.Finding {
		limit := limits.forFile(fc)
		lines := countLines(src)
		if lines <= limit {
			return nil
		}
		return []finding.Finding{reporter.At(
			fc.Path,
			0,
			fmt.Sprintf("file is %d lines, max %d", lines, limit),
			fmt.Sprintf("split into smaller files (max %d lines per file)", limit),
		)}
	})
}

func (analyzer *Analyzer) resolveMaxLines(ctx *pass.Context) int {
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

func (limits lineLimits) forFile(fc pass.FileChange) int {
	if fc.IsTest {
		return limits.test
	}
	return limits.production
}

func countLines(src []byte) int {
	return bytes.Count(src, []byte{'\n'})
}
