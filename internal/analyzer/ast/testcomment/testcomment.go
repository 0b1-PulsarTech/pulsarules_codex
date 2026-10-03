package testcomment

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

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

func checkFile(
	fset *token.FileSet, fc core.FileChange, f *ast.File, reporter core.Reporter,
) []core.Finding {
	var findings []core.Finding
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		kind, ok := gotest.TestKind(f, fn)
		if !ok {
			continue
		}
		if group, ok := nonExemptGroup(fn.Doc); ok {
			findings = append(findings, reporter.At(
				fc.Path,
				fset.Position(group.Pos()).Line,
				fmt.Sprintf(
					"%s carries a doc comment; the test name pins what it verifies",
					fn.Name.Name,
				),
				"delete it, or move a genuine precondition into a why: comment on a helper it "+
					"calls",
			))
		}
		for _, group := range bodyComments(f.Comments, fn) {
			if kind == gotest.KindExample && isExampleOutputComment(group) {
				continue
			}
			if inner, ok := nonExemptGroup(group); ok {
				findings = append(findings, reporter.At(
					fc.Path,
					fset.Position(inner.Pos()).Line,
					fmt.Sprintf(
						"comment inside %s narrates what the test already names",
						fn.Name.Name,
					),
					"delete it, or rename the test/sub-case so the name carries it",
				))
			}
		}
	}
	return findings
}

func isExampleOutputComment(group *ast.CommentGroup) bool {
	text := strings.ToLower(strings.TrimSpace(group.Text()))
	return strings.HasPrefix(text, "output:") || strings.HasPrefix(text, "unordered output:")
}

func nonExemptGroup(group *ast.CommentGroup) (*ast.CommentGroup, bool) {
	if group == nil {
		return nil, false
	}
	for _, c := range group.List {
		if !core.IsExemptHeaderLine(c.Text) {
			return group, true
		}
	}
	return nil, false
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
