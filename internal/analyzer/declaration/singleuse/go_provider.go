package singleuse

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type goProvider struct{}

var _ DeclarationProvider = goProvider{}

func declKindFor(tok token.Token) DeclKind {
	if tok == token.CONST {
		return DeclConst
	}
	return DeclVar
}

func (goProvider) Declarations(ctx *pass.Context) []Declaration {
	fset := ctx.ASTCache.FileSet()
	var out []Declaration
	for fc, file := range ctx.ChangedGoASTs() {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || (gen.Tok != token.CONST && gen.Tok != token.VAR) {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				out = append(out, valueDeclarations(gen, vs, fc, fset)...)
			}
		}
	}
	return out
}

func valueDeclarations(
	gen *ast.GenDecl, vs *ast.ValueSpec, fc pass.FileChange, fset *token.FileSet,
) []Declaration {
	typeName := typeString(vs.Type)
	annotations := docLines(gen.Doc, vs.Doc)
	var out []Declaration
	for nameIndex, name := range vs.Names {
		if name.Name == "_" {
			continue
		}
		out = append(out, Declaration{
			Name:        name.Name,
			Kind:        declKindFor(gen.Tok),
			Exported:    ast.IsExported(name.Name),
			TypeName:    typeName,
			Annotations: annotations,
			Init:        initKindFor(vs, nameIndex),
			File:        fc.Path,
			Line:        fset.Position(name.Pos()).Line,
		})
	}
	return out
}

func typeString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	return types.ExprString(expr)
}

func docLines(groups ...*ast.CommentGroup) []string {
	var lines []string
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			lines = append(lines, comment.Text)
		}
	}
	return lines
}

func initKindFor(vs *ast.ValueSpec, idx int) InitKind {
	var value ast.Expr
	switch {
	case len(vs.Values) == len(vs.Names):
		value = vs.Values[idx]
	case len(vs.Values) == 1:
		// The broadcast form var a, b = f().
		value = vs.Values[0]
	default:
		return InitNone
	}
	for {
		paren, ok := value.(*ast.ParenExpr)
		if !ok {
			break
		}
		value = paren.X
	}
	switch value.(type) {
	case *ast.CallExpr:
		return InitCall
	case *ast.CompositeLit:
		return InitComposite
	default:
		return InitLiteral
	}
}
