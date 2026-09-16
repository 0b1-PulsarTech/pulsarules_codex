package singleuse

import (
	"go/ast"
	"go/token"
	"path"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func (goProvider) SoleUser(ctx *pass.Context, decl Declaration) (string, bool) {
	fset := ctx.ASTCache.FileSet()
	usedIn := make(map[ast.Node]bool)
	files, complete := packageFiles(ctx, path.Dir(decl.File))
	if !complete {
		return "", false
	}
	for _, file := range files {
		ast.Walk(&usageWalker{fset: fset, decl: decl, usedIn: usedIn}, file)
	}
	// The nil key is a package-level reference, which disqualifies the candidate.
	if usedIn[nil] || len(usedIn) != 1 {
		return "", false
	}
	var owner ast.Node
	for node := range usedIn {
		owner = node
	}
	return describeOwner(owner), true
}

func describeOwner(owner ast.Node) string {
	switch typed := owner.(type) {
	case *ast.FuncDecl:
		return typed.Name.Name
	case *ast.FuncLit:
		return "the enclosing closure"
	default:
		return "the enclosing function"
	}
}

type usageWalker struct {
	fset      *token.FileSet
	decl      Declaration
	enclosing ast.Node
	usedIn    map[ast.Node]bool
}

func (walker *usageWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if ident, ok := node.(*ast.Ident); ok {
		if ident.Name == walker.decl.Name && !walker.isDeclSite(ident) {
			walker.usedIn[walker.enclosing] = true
		}
		return nil
	}
	switch node.(type) {
	case *ast.FuncDecl, *ast.FuncLit:
		return &usageWalker{
			fset: walker.fset, decl: walker.decl, enclosing: node, usedIn: walker.usedIn,
		}
	}
	return walker
}

func (walker *usageWalker) isDeclSite(ident *ast.Ident) bool {
	// Matches by file and line: Go forbids redeclaring a name there.
	pos := walker.fset.Position(ident.Pos())
	return pos.Filename == walker.decl.File && pos.Line == walker.decl.Line
}

func packageFiles(ctx *pass.Context, dir string) ([]*ast.File, bool) {
	var files []*ast.File
	// A missed file hides references, so a used declaration would read as single-use.
	complete := true
	ctx.Sources.Walk(func(srcPath, ext string) bool {
		if ext != ".go" || path.Dir(srcPath) != dir {
			return true
		}
		src, ok := ctx.Sources.Read(srcPath)
		if !ok {
			complete = false
			return true
		}
		// Reparsing through the shared cache keeps positions comparable.
		file, err := ctx.ASTCache.Parse(srcPath, src)
		if err != nil {
			complete = false
			return true
		}
		files = append(files, file)
		return true
	})
	return files, complete
}
