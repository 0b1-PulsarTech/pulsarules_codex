package packagevar

import (
	"go/ast"
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

const ID = "package-level-var"

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
	}
}

func (analyzer *Analyzer) ID() string             { return ID }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	reported := alreadyReported(ctx)
	dirScans := scanDirs(ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		dir := dirOf(fc.Path)
		if fc.Generated || !gomod.IsPackageDir(dir) {
			return nil
		}
		siblings := dirScans.dirs[pkgKey{dir: dir, pkg: file.Name.Name}]
		imports := goast.AllImportNames(file)
		scan := fileScan{
			fset:     ctx.ASTCache.FileSet(),
			path:     fc.Path,
			file:     file,
			reporter: reporter,
			reported: reported,
			imports:  imports,
			types:    siblings.types,
			errFuncs: siblings.errFuncs,
			embeds:   embedGroups(file),
			enums:    enumNames(file, imports, siblings.consts),
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

func alreadyReported(ctx *pass.Context) map[site]bool {
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
