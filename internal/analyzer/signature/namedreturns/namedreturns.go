package namedreturns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "named-returns"

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	return ctx.EachGoFile(func(fc pass.FileChange, file *ast.File) []finding.Finding {
		return checkFile(ctx.ASTCache.FileSet(), fc, file, analyzer.reporter)
	})
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding

	ast.Inspect(file, func(node ast.Node) bool {
		var sites []goast.FuncSite
		switch typed := node.(type) {
		case *ast.FuncDecl:
			sites = []goast.FuncSite{goast.FuncDeclSite(typed)}
		case *ast.InterfaceType:
			sites = goast.InterfaceMethodSites(typed)
		}
		for _, site := range sites {
			findings = append(findings, checkResults(fset, fc, site, reporter)...)
		}
		return true
	})

	return findings
}

func checkResults(
	fset *token.FileSet, fc pass.FileChange, site goast.FuncSite, reporter finding.Reporter,
) []finding.Finding {
	dup, found := duplicateResultType(site.Results)
	if !found {
		return nil
	}
	return []finding.Finding{reporter.At(
		fc.Path,
		fset.Position(site.Pos).Line,
		fmt.Sprintf(
			"%s returns more than one unnamed %s, so only the order tells them apart",
			site.Name,
			dup,
		),
		"name the results so the signature says which is which",
	)}
}

func duplicateResultType(results *ast.FieldList) (string, bool) {
	if results == nil || len(results.List) < 2 {
		return "", false
	}

	seen := make(map[string]bool, len(results.List))
	for _, field := range results.List {
		// Naming even one result names them all, so a named signature is left alone.
		if len(field.Names) > 0 {
			return "", false
		}
		// types.ExprString reads the type off the syntax alone, so no package
		// has to be loaded to compare two result types.
		name := types.ExprString(field.Type)
		if seen[name] {
			return name, true
		}
		seen[name] = true
	}
	return "", false
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "named-returns"}
}
