package filesize

import (
	"bytes"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

const ID = "file-size"

type Analyzer struct {
	maxLines int
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	const defaultMaxLines = 180
	return &Analyzer{
		maxLines: defaultMaxLines,
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatProject),
	}
}

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	warning := pass.ResolveReporter(analyzer.reporter, ctx)
	// A severity override on file-size downgrades the cap too, since both share one ID.
	hardCap := finding.NewReporter(ID, finding.SeverityError, finding.CatProject)
	eligible := pass.ExtensionFilter(ctx, ID, language.SourceExtensions())
	return pass.EachChangedFile(
		ctx,
		eligible,
		func(fc pass.FileChange, src []byte) []finding.Finding {
			params := ctx.ParamsAt(ID, fc.Path)
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

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "source-files",
		Params: []string{ParamMaxLines, ParamErrorLines, pass.ParamExtensions},
	}
}
