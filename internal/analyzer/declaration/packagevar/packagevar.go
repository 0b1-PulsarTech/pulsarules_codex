package packagevar

import (
	"go/ast"
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "package-level-var"

type Analyzer struct {
	reporter pass.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: pass.NewReporter(ID, pass.SeverityWarning, pass.CatAST),
	}
}

func (analyzer *Analyzer) ID() string          { return ID }
func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	reported := alreadyReported(ctx)
	dirScans := scanDirs(ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []pass.Finding {
		dir := dirOf(fc.Path)
		if fc.Generated || isNonPackageDir(dir) {
			return nil
		}
		siblings := dirScans.dirs[pkgKey{dir: dir, pkg: file.Name.Name}]
		scan := fileScan{
			fset:      ctx.ASTCache.FileSet(),
			path:      fc.Path,
			file:      file,
			reporter:  reporter,
			reported:  reported,
			errorsPkg: importName(file, "errors"),
			fmtPkg:    importName(file, "fmt"),
			imports:   importNames(file),
			types:     siblings.types,
			errFuncs:  siblings.errFuncs,
			embeds:    embedGroups(file),
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
