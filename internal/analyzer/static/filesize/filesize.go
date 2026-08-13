package filesize

import (
	"bytes"
	"fmt"
	"math"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const (
	defaultMaxLines = 180

	// testLineMultiplier is the allowance a _test.go file gets over the
	// limit for production code. A table-driven test carries a case table
	// that has no counterpart in the code under test, so holding both to
	// one number would push the table out of the file it documents.
	testLineMultiplier = 2.6
)

// Analyzer reports Go source files that exceed a configurable line count
// threshold. The default max is 180 lines, and a _test.go file is allowed
// testLineMultiplier times that; 0 disables the check.
type Analyzer struct {
	maxLines int
}

var fileSizeReporter = core.NewReporter("file-size", core.SeverityWarning, core.CatProject)

func NewAnalyzer() *Analyzer {
	return &Analyzer{maxLines: defaultMaxLines}
}

func (a *Analyzer) ID() string   { return "file-size" }
func (a *Analyzer) Name() string { return "File size" }
func (a *Analyzer) Description() string {
	return "Reports Go files that exceed the configured line count"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageStatic }
func (a *Analyzer) Category() core.Category { return core.CatProject }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{}
}

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
			findings = append(findings, fileSizeReporter.At(
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

// lineLimits holds both thresholds one run applies. It is derived once from
// max_lines, so raising or lowering that carries the test allowance with it
// instead of leaving a second number to keep in sync.
type lineLimits struct {
	production int
	test       int
}

func newLineLimits(maxLines int) lineLimits {
	return lineLimits{
		production: maxLines,
		test:       int(math.Round(float64(maxLines) * testLineMultiplier)),
	}
}

// forFile reads as a lookup on the file rather than a boolean handed to a
// function, which is what the flag-argument smell is about.
func (l lineLimits) forFile(fc core.FileChange) int {
	if fc.IsTest {
		return l.test
	}
	return l.production
}

func countLines(src []byte) int {
	return bytes.Count(src, []byte{'\n'})
}
