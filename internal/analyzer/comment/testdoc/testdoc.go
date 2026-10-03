package testdoc

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast/testfunc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string { return "test-comment" }

func (a *Analyzer) Stage() pass.StageID { return pass.StageAST }

func (a *Analyzer) reporter() pass.Reporter {
	return pass.NewReporter(a.ID(), pass.SeverityWarning, pass.CatAST)
}

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	reporter := pass.ResolveReporter(a.reporter(), ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []pass.Finding {
		if !fc.IsTest || fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

type fileScan struct {
	fset     *token.FileSet
	fc       pass.FileChange
	file     *ast.File
	reporter pass.Reporter
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, f *ast.File, reporter pass.Reporter,
) []pass.Finding {
	scan := fileScan{fset: fset, fc: fc, file: f, reporter: reporter}
	var findings []pass.Finding
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !testfunc.Covers(fc, f, fn) {
			continue
		}
		if !goast.IsDirectiveGroup(fn.Doc) {
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(fn.Doc.Pos()).Line,
				fmt.Sprintf(
					"%s carries a doc comment; test functions and helpers carry no comments",
					fn.Name.Name,
				),
				goast.HintFor(fn.Doc, "delete it; the name carries what it does"),
			))
		}
		findings = append(findings, scan.body(fn)...)
	}
	return findings
}

func (scan fileScan) body(fn *ast.FuncDecl) []pass.Finding {
	kind, _ := testfunc.Kind(scan.file, fn)
	var findings []pass.Finding
	for _, group := range bodyComments(scan.file.Comments, fn) {
		if kind == testfunc.KindExample && isExampleOutputComment(group) {
			continue
		}
		if goast.IsDirectiveGroup(group) {
			continue
		}
		findings = append(findings, scan.reporter.At(
			scan.fc.Path,
			scan.fset.Position(group.Pos()).Line,
			fmt.Sprintf(
				"comment inside %s; test functions and helpers carry no comments",
				fn.Name.Name,
			),
			goast.HintFor(group, "delete it; the name carries what it does"),
		))
	}
	return findings
}

func isExampleOutputComment(group *ast.CommentGroup) bool {
	text := strings.ToLower(strings.TrimSpace(group.Text()))
	return strings.HasPrefix(text, "output:") || strings.HasPrefix(text, "unordered output:")
}

func bodyComments(all []*ast.CommentGroup, fn *ast.FuncDecl) []*ast.CommentGroup {
	if fn.Body == nil {
		return nil
	}
	var groups []*ast.CommentGroup
	for _, group := range all {
		if group.Pos() > fn.Body.Pos() && group.End() < fn.Body.End() {
			groups = append(groups, group)
		}
	}
	return groups
}
