package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func (coll *collector) docStatements(statements []*ast.Node, owner string) {
	for _, stmt := range statements {
		coll.docDeclaration(stmt, owner)
	}
}

func (coll *collector) docDeclaration(node *ast.Node, owner string) {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		coll.docSite(node, node.Name(), syntax.DocFunction, owner)
	case ast.KindClassDeclaration, ast.KindInterfaceDeclaration, ast.KindEnumDeclaration:
		coll.docType(node, owner, node.Members())
	case ast.KindTypeAliasDeclaration:
		coll.docType(node, owner, nil)
	case ast.KindModuleDeclaration:
		coll.docModule(node, owner)
	case ast.KindVariableStatement:
		coll.docVariable(node, owner)
	default:
		coll.docMember(node, owner)
	}
}

func (coll *collector) docType(node *ast.Node, owner string, members []*ast.Node) {
	coll.docSite(node, node.Name(), syntax.DocType, owner)
	scope := qualified(owner, coll.nameOr(node.Name(), ""))
	coll.docStatements(members, scope)
}

func (coll *collector) docModule(node *ast.Node, owner string) {
	coll.docSite(node, node.Name(), syntax.DocType, owner)
	body := node.Body()
	if body == nil {
		return
	}
	scope := qualified(owner, coll.nameOr(node.Name(), ""))
	if body.Kind == ast.KindModuleBlock {
		coll.docStatements(body.Statements(), scope)
		return
	}
	coll.docDeclaration(body, scope)
}

func (coll *collector) docMember(node *ast.Node, owner string) {
	switch node.Kind {
	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindGetAccessor,
		ast.KindSetAccessor:
		coll.docSite(node, node.Name(), syntax.DocMethod, owner)
	case ast.KindConstructor:
		coll.docSiteNamed(node, "constructor", syntax.DocConstructor, owner)
	case ast.KindPropertyDeclaration, ast.KindPropertySignature, ast.KindEnumMember:
		coll.docSite(node, node.Name(), syntax.DocField, owner)
	}
}

func (coll *collector) docVariable(node *ast.Node, owner string) {
	declarations := node.AsVariableStatement().DeclarationList.AsVariableDeclarationList()
	if len(declarations.Declarations.Nodes) == 0 {
		return
	}
	first := declarations.Declarations.Nodes[0]
	kind := syntax.DocVariable
	if isFunctionValue(first.Initializer()) {
		kind = syntax.DocFunction
	}
	coll.docSite(node, first.Name(), kind, owner)
}

func isFunctionValue(value *ast.Node) bool {
	for value != nil && isWrapper(value.Kind) {
		value = value.Expression()
	}
	return value != nil &&
		(value.Kind == ast.KindArrowFunction || value.Kind == ast.KindFunctionExpression)
}

func (coll *collector) docSite(node, name *ast.Node, kind syntax.DocKind, owner string) {
	coll.docSiteNamed(node, coll.nameOr(name, ""), kind, owner)
}

func (coll *collector) docSiteNamed(
	node *ast.Node, name string, kind syntax.DocKind, owner string,
) {
	doc, found := coll.docComment(node)
	if !found {
		return
	}
	coll.out.DocSites = append(coll.out.DocSites, syntax.DocSite{
		Symbol:         qualified(owner, name),
		Kind:           kind,
		Line:           coll.lines.Of(doc.start),
		Exempt:         doc.exempt,
		MixedDirective: doc.mixed,
	})
}

func qualified(owner, name string) string {
	if owner == "" {
		return name
	}
	return owner + "." + name
}
