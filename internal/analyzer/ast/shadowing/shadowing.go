package shadowing

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// Analyzer detects variable and builtin shadowing in Go functions.
// It walks each function body with a scope stack and flags any declaration
// whose name is already visible from an enclosing scope.
type Analyzer struct{}

var shadowingReporter = core.NewReporter("shadowing", core.SeverityWarning, core.CatAST)

// NewAnalyzer creates a shadowing analyzer.
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
	if ctx.ASTCache == nil {
		return nil
	}

	fset := ctx.ASTCache.FileSet()
	var findings []core.Finding
	for fc, f := range ctx.ChangedGoASTs() {
		findings = append(findings, a.checkFile(fset, fc, f)...)
	}
	return findings
}

// shadowAllowlist are names that are conventionally shadowed in idiomatic
// Go: ok/err in type assertions, found in range loops. Flagging these
// produces false positives on standard Go patterns.
var shadowAllowlist = map[string]bool{
	"ok": true,
}

func (a *Analyzer) checkFile(
	fset *token.FileSet,
	fc core.FileChange,
	f *ast.File,
) []core.Finding {
	var findings []core.Finding

	emit := func(pos token.Pos, name, kind string) {
		if shadowAllowlist[name] {
			return
		}
		findings = append(findings, shadowingReporter.At(
			fc.Path,
			fset.Position(pos).Line,
			fmt.Sprintf("%q shadows an outer %s", name, kind),
			fmt.Sprintf("rename the inner %q to avoid shadowing", name),
		))
	}

	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		scopes := newScopeStack()
		scopes.push() // function-level scope

		if fn.Name != nil {
			scopes.declare(fn.Name.Name, fn.Name.NamePos)
		}
		if fn.Type.Params != nil {
			for _, p := range fn.Type.Params.List {
				for _, id := range p.Names {
					scopes.declare(id.Name, id.NamePos)
				}
			}
		}
		if fn.Type.Results != nil {
			for _, r := range fn.Type.Results.List {
				for _, id := range r.Names {
					scopes.declare(id.Name, id.NamePos)
				}
			}
		}
		a.walkBlock(fn.Body, scopes, emit)
	}

	return findings
}
