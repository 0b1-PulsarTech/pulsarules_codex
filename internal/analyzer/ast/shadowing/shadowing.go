package shadowing

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer detects variable/builtin shadowing: names already visible from
// an enclosing scope, plus a `:=` that silently reassigns a signature-bound name.
//
// Known limit: function literals aren't walked - doing so mostly finds
// `if err := f(); err != nil` in closures, not worth reporting yet.
type Analyzer struct{}

var (
	shadowingReporter = core.NewReporter("shadowing", core.SeverityWarning, core.CatAST)
	// Same walk, second identity: the two findings share every bit of scope
	// machinery but say opposite things, and a reader who cannot tell them
	// apart cannot act on either. The commit analyzer splits its sub-rules the
	// same way.
	reuseReporter = core.NewReporter("short-decl-reuse", core.SeverityWarning, core.CatAST)
)

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) ID() string   { return "shadowing" }
func (a *Analyzer) Name() string { return "Variable shadowing" }
func (a *Analyzer) Description() string {
	return "Reports names that shadow a name from an enclosing scope or a Go builtin"
}
func (a *Analyzer) Stage() core.StageID     { return core.StageAST }
func (a *Analyzer) Category() core.Category { return core.CatAST }
func (a *Analyzer) Needs() core.Requirements {
	return core.Requirements{NeedsAST: true}
}

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	return core.RunPerGoFile(ctx, func(fc core.FileChange, f *ast.File) []core.Finding {
		return a.checkFile(ctx.ASTCache.FileSet(), fc, f)
	})
}

// shadowAllowlist are names that are conventionally shadowed in idiomatic
// Go: ok in type assertions. Flagging these produces false positives on
// standard Go patterns. It gates shadowing only: a signature name reused by
// `:=` is never a coincidence, so short-decl-reuse reports it regardless.
var shadowAllowlist = map[string]bool{
	"ok": true,
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	fc core.FileChange,
	f *ast.File,
) []core.Finding {
	var findings []core.Finding
	// One report per name per function. The mandated `(out T, err error)` plus
	// a commit/rollback defer makes reusing err routine, and repeating the
	// same advice at every `:=` in the body buries the other findings.
	reported := make(map[string]bool)

	report := reporter{
		shadow: func(pos token.Pos, name, kind string) {
			if shadowAllowlist[name] {
				return
			}
			findings = append(findings, shadowingReporter.At(
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
			findings = append(findings, reuseReporter.At(
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

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		scopes := newScopeStack()
		clear(reported)

		// Package scope: the function's own name is not part of its block.
		scopes.push()
		if fn.Name != nil {
			scopes.declare(fn.Name.Name)
		}

		// The function block. Go puts the receiver, the parameters, the named
		// results AND the body's top-level statements in this one block, so
		// walkStmts runs the body here instead of pushing a block of its own.
		scopes.push()
		declareFields(scopes, fn.Recv, fn.Type.Params, fn.Type.Results)
		a.walkStmts(fn.Body.List, scopes, report)
	}

	return findings
}

// declareFields binds a signature's names into the CURRENT scope, which for a
// function or a closure is the same block its body statements live in.
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
