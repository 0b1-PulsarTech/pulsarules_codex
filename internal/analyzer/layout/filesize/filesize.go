package filesize

import (
	"bytes"
	"fmt"

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
	warning := pass.ResolveReporter(analyzer.reporter, ctx)
	// A severity override on file-size downgrades the cap too, since both share one ID.
	hardCap := finding.NewReporter(analyzer.ID(), finding.SeverityError, finding.CatProject)
	eligible := pass.ExtensionFilter(ctx, analyzer.ID(), pass.DefaultSourceExtensions())
	params := ctx.Params(analyzer.ID())
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			limits := newLineLimits(
				params.Int(ParamMaxLines, analyzer.maxLines), errorLines(params, fc.Extension),
			)
			lines := countLines(src)
			limit, hardLimit := limits.forFile(fc), limits.hardFor(fc)
			switch {
			case hardLimit > 0 && lines > hardLimit:
				return []finding.Finding{
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
			return []finding.Finding{warning.At(
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
