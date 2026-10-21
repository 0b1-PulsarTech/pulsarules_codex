package testdoc

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/goast/testfunc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "test-comment"

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) reporter() finding.Reporter {
	return finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST)
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	reporter := pass.ResolveReporter(analyzer.reporter(), ctx)
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, file *ast.File) []finding.Finding {
		if !fc.IsTest || fc.Generated {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, file, reporter)
	})
}

type fileScan struct {
	fset     *token.FileSet
	fc       pass.FileChange
	file     *ast.File
	reporter finding.Reporter
}

func checkFile(
	fset *token.FileSet, fc pass.FileChange, file *ast.File, reporter finding.Reporter,
) []finding.Finding {
	scan := fileScan{fset: fset, fc: fc, file: file, reporter: reporter}
	var findings []finding.Finding
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !testfunc.Covers(fc, file, fn) {
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

func (scan fileScan) body(fn *ast.FuncDecl) []finding.Finding {
	kind, _ := testfunc.Kind(scan.file, fn)
	var findings []finding.Finding
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

func Describe() pass.Descriptor {
	return pass.Descriptor{ID: ID, Rule: "comments"}
}
