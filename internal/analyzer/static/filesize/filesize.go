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
	if ctx.Sources == nil {
		return nil
	}
	var findings []core.Finding
	for _, fc := range ctx.ChangedFiles {
		if fc.Extension != ".go" {
			continue
		}
		src, ok := ctx.Sources.Read(fc.Path)
		if !ok {
			continue
		}
		limit := limits.forFile(fc)
		lines := countLines(src)
		if lines > limit {
			findings = append(findings, a.reporter.At(
				fc.Path,
				0,
				fmt.Sprintf("file is %d lines, max %d", lines, limit),
				fmt.Sprintf("split into smaller files (max %d lines per file)", limit),
			))
		}
	}
	return findings
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

func (l lineLimits) forFile(fc core.FileChange) int {
	if fc.IsTest {
		return l.test
	}
	return l.production
}

func countLines(src []byte) int {
	return bytes.Count(src, []byte{'\n'})
}
