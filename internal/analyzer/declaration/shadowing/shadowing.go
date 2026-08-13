package shadowing

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type Analyzer struct {
	shadowReporter finding.Reporter
	reuseReporter  finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		shadowReporter: finding.NewReporter("shadowing", finding.SeverityWarning, finding.CatAST),
		// The two findings share scope machinery but say opposite things.
		reuseReporter: finding.NewReporter(
			"short-decl-reuse",
			finding.SeverityWarning,
			finding.CatAST,
		),
	}
}

func (a *Analyzer) ID() string             { return "shadowing" }
func (a *Analyzer) Stage() finding.StageID { return finding.StageAST }
func (a *Analyzer) Needs() finding.Requirements {
	return finding.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	return pass.RunPerGoFile(ctx, func(fc pass.FileChange, f *ast.File) []finding.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

func (a *Analyzer) forFunc(
	fset *token.FileSet, fc pass.FileChange, findings *[]finding.Finding,
) reporter {
	reported := make(map[string]bool)
	// One report per name per function. The mandated `(out T, err error)` plus
	// a commit/rollback defer makes reusing err routine, and repeating the
	// same advice at every `:=` in the body buries the other findings.
	return reporter{
		shadow: func(pos token.Pos, name, kind string) {
			if name == "ok" {
				return
			}
			*findings = append(*findings, a.shadowReporter.At(
				fc.Path,
				fset.Position(pos).Line,
				fmt.Sprintf("%q shadows an outer %s", name, kind),
				fmt.Sprintf("rename the inner %q to avoid shadowing", name),
			))
		},
		reuse: func(pos token.Pos, name string) {
			if reported[name] {
				return
			}
			reported[name] = true
			*findings = append(*findings, a.reuseReporter.At(
				fc.Path,
				fset.Position(pos).Line,
				fmt.Sprintf("%q is reassigned by := in the block that declares it", name),
				fmt.Sprintf(
					"declare the new variables with var above and assign with =, "+
						"so reusing %q reads as the reassignment it is",
					name,
				),
			))
		},
	}
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	fc pass.FileChange,
	f *ast.File,
) []finding.Finding {
	var findings []finding.Finding
	// simplification: function literals are not walked; doing so mostly finds
	// `if err := f(); err != nil` in closures, not worth reporting yet. Upgrade
	// path: walk FuncLit bodies in walkStmt.
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		scopes := newScopeStack()
		report := a.forFunc(fset, fc, &findings)

		// Package scope: the function's own name is not part of its block.
		scopes.push()
		if fn.Name != nil {
			scopes.declare(fn.Name.Name)
		}

		// The function block. Go puts the receiver, the parameters, the named
		// results and the body's top-level statements in this one block, so
		// walkStmts runs the body here instead of pushing a block of its own.
		scopes.push()
		declareFields(scopes, fn.Recv, fn.Type.Params, fn.Type.Results)
		a.walkStmts(fn.Body.List, scopes, report)
	}

	return findings
}

func declareFields(scopes *scopeStack, lists ...*ast.FieldList) {
	for _, list := range lists {
		if list == nil {
			continue
		}
		for _, field := range list.List {
			for _, id := range field.Names {
				if id.Name == "_" {
					continue
				}
				scopes.declareAs(id.Name, originSignature)
			}
		}
	}
}
