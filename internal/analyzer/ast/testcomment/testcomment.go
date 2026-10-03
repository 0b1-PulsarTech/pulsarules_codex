package testcomment

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/directive"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/ast/gotest"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string { return "test-comment" }

func (a *Analyzer) Stage() core.StageID { return core.StageAST }

func (a *Analyzer) reporter() core.Reporter {
	return core.NewReporter(a.ID(), core.SeverityWarning, core.CatAST)
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	reporter := a.reporter().Resolved(ctx)
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		if !fc.IsTest {
			return nil
		}
		return checkFile(ctx.ASTCache.FileSet(), fc, f, reporter)
	})
}

type fileScan struct {
	fset     *token.FileSet
	fc       core.FileChange
	file     *ast.File
	reporter core.Reporter
}

func checkFile(
	fset *token.FileSet, fc core.FileChange, f *ast.File, reporter core.Reporter,
) []core.Finding {
	scan := fileScan{fset: fset, fc: fc, file: f, reporter: reporter}
	var findings []core.Finding
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !gotest.CoversFunc(fc, f, fn) {
			continue
		}
		if !directive.IsDirectiveGroup(fn.Doc) {
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(fn.Doc.Pos()).Line,
				fmt.Sprintf(
					"%s carries a doc comment; test functions and helpers carry no comments",
					fn.Name.Name,
				),
				directive.HintFor(fn.Doc, "delete it; the name carries what it does"),
			))
		}
		findings = append(findings, scan.body(fn)...)
	}
	return findings
}

func (scan fileScan) body(fn *ast.FuncDecl) []core.Finding {
	kind, _ := gotest.TestKind(scan.file, fn)
	var findings []core.Finding
	for _, group := range bodyComments(scan.file.Comments, fn) {
		if kind == gotest.KindExample && isExampleOutputComment(group) {
			continue
		}
		if directive.IsDirectiveGroup(group) {
			continue
		}
		findings = append(findings, scan.reporter.At(
			scan.fc.Path,
			scan.fset.Position(group.Pos()).Line,
			fmt.Sprintf(
				"comment inside %s; test functions and helpers carry no comments",
				fn.Name.Name,
			),
			directive.HintFor(group, "delete it; the name carries what it does"),
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
