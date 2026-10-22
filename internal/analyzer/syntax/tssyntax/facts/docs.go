package facts

import (
	"slices"
	"strings"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/scanner"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/jsdoc"
)

type docFacts struct {
	source
	factory *ast.NodeFactory
	out     []syntax.DocSite
}

func (docs *docFacts) docStatements(statements []*ast.Node, owner string) {
	for _, stmt := range statements {
		docs.docDeclaration(stmt, owner)
	}
}

func (docs *docFacts) docDeclaration(node *ast.Node, owner string) {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		docs.docSite(node, node.Name(), syntax.DocFunction, owner)
	case ast.KindClassDeclaration, ast.KindInterfaceDeclaration, ast.KindEnumDeclaration:
		docs.docType(node, owner, node.Members())
	case ast.KindTypeAliasDeclaration:
		docs.docType(node, owner, nil)
	case ast.KindModuleDeclaration:
		docs.docModule(node, owner)
	case ast.KindVariableStatement:
		docs.docVariable(node, owner)
	default:
		docs.docMember(node, owner)
	}
}

func (docs *docFacts) docType(node *ast.Node, owner string, members []*ast.Node) {
	docs.docSite(node, node.Name(), syntax.DocType, owner)
	scope := qualified(owner, docs.nameOr(node.Name(), ""))
	docs.docStatements(members, scope)
}

func (docs *docFacts) docModule(node *ast.Node, owner string) {
	docs.docSite(node, node.Name(), syntax.DocType, owner)
	body := node.Body()
	if body == nil {
		return
	}
	scope := qualified(owner, docs.nameOr(node.Name(), ""))
	if body.Kind == ast.KindModuleBlock {
		docs.docStatements(body.Statements(), scope)
		return
	}
	docs.docDeclaration(body, scope)
}

func (docs *docFacts) docMember(node *ast.Node, owner string) {
	switch node.Kind {
	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindGetAccessor,
		ast.KindSetAccessor:
		docs.docSite(node, node.Name(), syntax.DocMethod, owner)
	case ast.KindConstructor:
		docs.docSiteNamed(node, "constructor", syntax.DocConstructor, owner)
	case ast.KindPropertyDeclaration, ast.KindPropertySignature, ast.KindEnumMember:
		docs.docSite(node, node.Name(), syntax.DocField, owner)
	}
}

func (docs *docFacts) docVariable(node *ast.Node, owner string) {
	declarations := node.AsVariableStatement().DeclarationList.AsVariableDeclarationList()
	if len(declarations.Declarations.Nodes) == 0 {
		return
	}
	first := declarations.Declarations.Nodes[0]
	kind := syntax.DocVariable
	if isFunctionValue(first.Initializer()) {
		kind = syntax.DocFunction
	}
	docs.docSite(node, first.Name(), kind, owner)
}

func isFunctionValue(value *ast.Node) bool {
	for value != nil && isWrapper(value.Kind) {
		value = value.Expression()
	}
	return value != nil &&
		(value.Kind == ast.KindArrowFunction || value.Kind == ast.KindFunctionExpression)
}

func (docs *docFacts) docSite(node, name *ast.Node, kind syntax.DocKind, owner string) {
	docs.docSiteNamed(node, docs.nameOr(name, ""), kind, owner)
}

func (docs *docFacts) docSiteNamed(
	node *ast.Node, name string, kind syntax.DocKind, owner string,
) {
	doc, found := docs.docComment(node)
	if !found {
		return
	}
	docs.out = append(docs.out, syntax.DocSite{
		Symbol:         qualified(owner, name),
		Kind:           kind,
		Line:           docs.lines.Of(doc.start),
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

type docShape struct {
	start         int
	exempt, mixed bool
}

func (docs *docFacts) docComment(node *ast.Node) (docShape, bool) {
	ranges := slices.Collect(
		scanner.GetLeadingCommentRanges(docs.factory, docs.text, node.Pos()),
	)
	boundary := scanner.SkipTrivia(docs.text, node.Pos())
	first := len(ranges)
	for first > 0 && strings.Count(docs.text[ranges[first-1].End():boundary], "\n") <= 1 {
		first--
		boundary = ranges[first].Pos()
	}
	chain := ranges[first:]
	doc := slices.IndexFunc(chain, func(comment ast.CommentRange) bool {
		return jsdoc.IsBlock(docs.text[comment.Pos():comment.End()])
	})
	if doc < 0 {
		return docShape{}, false
	}
	for next := doc + 1; next < len(chain); next++ {
		if jsdoc.IsBlock(docs.text[chain[next].Pos():chain[next].End()]) {
			doc = next
		}
	}
	return docs.shapeOf(chain, doc), true
}

func (docs *docFacts) shapeOf(chain []ast.CommentRange, doc int) docShape {
	var others []string
	for idx, comment := range chain {
		if idx != doc {
			others = append(others, docs.text[comment.Pos():comment.End()])
		}
	}
	shape := jsdoc.Classify(docs.text[chain[doc].Pos():chain[doc].End()], others)
	return docShape{start: chain[doc].Pos(), exempt: shape.Exempt, mixed: shape.Mixed}
}
