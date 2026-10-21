package packagevar

import (
	"go/ast"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/declaration/singleuse"
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

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := analyzer.reporter
	dirScans := scanDirs(ctx)
	return ctx.EachGoFile(func(fc pass.FileChange, file *ast.File) []finding.Finding {
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
	file   string
	line   int
	symbol string
}

func DropSingleUseDuplicates(findings []finding.Finding) []finding.Finding {
	reported := make(map[site]bool)
	for _, finding := range findings {
		if finding.AnalyzerID == singleuse.ID {
			reported[site{finding.File, finding.Line, finding.Symbol}] = true
		}
	}
	return slices.DeleteFunc(findings, func(finding finding.Finding) bool {
		return finding.AnalyzerID == ID &&
			reported[site{finding.File, finding.Line, finding.Symbol}]
	})
}

func isTestFile(file string) bool { return strings.HasSuffix(file, "_test.go") }

func dirOf(file string) string { return path.Dir(filepath.ToSlash(file)) }

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "code-smells"}
}
