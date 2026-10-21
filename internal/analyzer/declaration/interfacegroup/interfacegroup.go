package interfacegroup

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "ungrouped-interfaces"

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(
		finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
		ctx,
	)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		if fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

type interfaceDecl struct {
	decl  *ast.GenDecl
	count int
	first *ast.TypeSpec
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter finding.Reporter,
) []finding.Finding {
	decls := interfaceDecls(file)
	total := 0
	for _, decl := range decls {
		total += decl.count
	}
	if total < 2 {
		return nil
	}

	main := mainGroup(decls)
	var first *ast.TypeSpec
	outside := 0
	for _, decl := range decls {
		if decl.decl == main {
			continue
		}
		if first == nil {
			first = decl.first
		}
		outside += decl.count
	}
	if first == nil {
		return nil
	}
	noun := "interfaces"
	if outside == 1 {
		noun = "interface"
	}
	return []finding.Finding{
		reporter.At(
			fc.Path,
			fset.Position(first.Pos()).Line,
			fmt.Sprintf("file declares %d %s outside a single type (...) block", outside, noun),
			"group every interface of the file in one type ( ... ) block",
		),
	}
}

func interfaceDecls(file *ast.File) []interfaceDecl {
	var decls []interfaceDecl
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		found := interfaceDecl{decl: gen}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			_, isInterface := typeSpec.Type.(*ast.InterfaceType)
			if !isInterface || typeSpec.Assign.IsValid() {
				continue
			}
			if found.count == 0 {
				found.first = typeSpec
			}
			found.count++
		}
		if found.count > 0 {
			decls = append(decls, found)
		}
	}
	return decls
}

func mainGroup(decls []interfaceDecl) *ast.GenDecl {
	var best *ast.GenDecl
	bestCount := 0
	for _, decl := range decls {
		if decl.decl.Lparen.IsValid() && decl.count > bestCount {
			best, bestCount = decl.decl, decl.count
		}
	}
	return best
}

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "effective-go"}
}
