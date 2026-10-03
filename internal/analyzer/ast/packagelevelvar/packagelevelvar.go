package packagelevelvar

import (
	"go/ast"
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const analyzerID = "package-level-var"

type Analyzer struct {
	reporter core.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: core.NewReporter(analyzerID, core.SeverityWarning, core.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return analyzerID }
func (analyzer *Analyzer) Stage() core.StageID { return core.StageAST }

func (analyzer *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := analyzer.reporter.Resolved(ctx)
	reported := alreadyReported(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		dir := dirOf(fc.Path)
		if fc.Generated || isNonPackageDir(dir) {
			return nil
		}
		scan := fileScan{
			fset:     ctx.ASTCache.FileSet(),
			path:     fc.Path,
			file:     file,
			reporter: reporter,
			reported: reported,
		}
		return scan.run()
	})
}

type site struct {
	file string
	line int
	name string
}

func alreadyReported(ctx *core.AnalysisContext) map[site]bool {
	reported := make(map[site]bool)
	// single-use-declaration runs first; skipping its sites avoids a double report.
	for _, finding := range ctx.Findings {
		if finding.AnalyzerID != "single-use-declaration" {
			continue
		}
		reported[site{file: finding.File, line: finding.Line, name: finding.Symbol}] = true
	}
	return reported
}

func dirOf(file string) string { return path.Dir(filepath.ToSlash(file)) }

func isNonPackageDir(dir string) bool {
	if dir == "." {
		return false
	}
	for segment := range strings.SplitSeq(dir, "/") {
		if strings.HasPrefix(segment, ".") || strings.HasPrefix(segment, "_") ||
			segment == "testdata" {
			return true
		}
	}
	return false
}
