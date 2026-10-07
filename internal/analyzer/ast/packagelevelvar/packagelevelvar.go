package packagelevelvar

import (
	"go/ast"
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
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
	dirScans := scanDirs(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, file *ast.File) []core.Finding {
		dir := dirOf(fc.Path)
		if fc.Generated || sources.IsNonPackageDir(dir) {
			return nil
		}
		siblings := dirScans.dirs[pkgKey{dir: dir, pkg: file.Name.Name}]
		imports := importNames(file)
		scan := fileScan{
			fset:      ctx.ASTCache.FileSet(),
			path:      fc.Path,
			file:      file,
			reporter:  reporter,
			reported:  reported,
			errorsPkg: importName(file, "errors"),
			fmtPkg:    importName(file, "fmt"),
			imports:   imports,
			types:     siblings.types,
			errFuncs:  siblings.errFuncs,
			embeds:    embedGroups(file),
			enums:     enumNames(file, imports, siblings.consts),
		}
		if isTestFile(fc.Path) {
			scan.assigned = dirScans.assigned[pkgKey{dir: dir, pkg: file.Name.Name}]
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

func isTestFile(file string) bool { return strings.HasSuffix(file, "_test.go") }

func dirOf(file string) string { return path.Dir(filepath.ToSlash(file)) }
