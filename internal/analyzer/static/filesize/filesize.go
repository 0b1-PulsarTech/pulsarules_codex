package filesize

import (
	"bytes"
	"fmt"

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
	warning := analyzer.reporter.Resolved(ctx)
	// A severity override on file-size downgrades the cap too, since both share one ID.
	hardCap := core.NewReporter(analyzer.ID(), core.SeverityError, core.CatProject)
	eligible := core.ExtensionFilter(ctx, analyzer.ID(), core.DefaultSourceExtensions())
	params := ctx.Params(analyzer.ID())
	return core.EachChangedFile(
		ctx,
		eligible,
		func(fc core.FileChange, src []byte) []core.Finding {
			limits := newLineLimits(
				params.Int(ParamMaxLines, analyzer.maxLines), errorLines(params, fc.Extension),
			)
			lines := countLines(src)
			limit, hardLimit := limits.forFile(fc), limits.hardFor(fc)
			switch {
			case hardLimit > 0 && lines > hardLimit:
				return []core.Finding{
					hardCap.At(
						fc.Path,
						0,
						fmt.Sprintf("file is %d lines, hard cap %d", lines, hardLimit),
						fmt.Sprintf(
							"split into smaller files (hard cap %d lines per file)", hardLimit,
						),
					),
				}
			case lines <= limit:
				return nil
			}
			return []core.Finding{warning.At(
				fc.Path,
				0,
				fmt.Sprintf("file is %d lines, max %d", lines, limit),
				fmt.Sprintf("split into smaller files (max %d lines per file)", limit),
			)}
		},
	)
}

func countLines(src []byte) int {
	return bytes.Count(src, []byte{'\n'})
}
