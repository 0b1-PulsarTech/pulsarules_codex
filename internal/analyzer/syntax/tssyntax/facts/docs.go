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

func (facts *docFacts) docStatements(statements []*ast.Node, owner string) {
	for _, stmt := range statements {
		facts.docDeclaration(stmt, owner)
	}
}

func (facts *docFacts) docDeclaration(node *ast.Node, owner string) {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		facts.docSite(node, node.Name(), syntax.DocFunction, owner)
	case ast.KindClassDeclaration, ast.KindInterfaceDeclaration, ast.KindEnumDeclaration:
		facts.docType(node, owner, node.Members())
	case ast.KindTypeAliasDeclaration:
		facts.docType(node, owner, nil)
	case ast.KindModuleDeclaration:
		facts.docModule(node, owner)
	case ast.KindVariableStatement:
		facts.docVariable(node, owner)
	default:
		facts.docMember(node, owner)
	}
}

func (facts *docFacts) docType(node *ast.Node, owner string, members []*ast.Node) {
	facts.docSite(node, node.Name(), syntax.DocType, owner)
	scope := qualified(owner, facts.nameOr(node.Name(), ""))
	facts.docStatements(members, scope)
}

func (facts *docFacts) docModule(node *ast.Node, owner string) {
	facts.docSite(node, node.Name(), syntax.DocType, owner)
	body := node.Body()
	if body == nil {
		return
	}
	scope := qualified(owner, facts.nameOr(node.Name(), ""))
	if body.Kind == ast.KindModuleBlock {
		facts.docStatements(body.Statements(), scope)
		return
	}
	facts.docDeclaration(body, scope)
}

func (facts *docFacts) docMember(node *ast.Node, owner string) {
	switch node.Kind {
	case ast.KindMethodDeclaration, ast.KindMethodSignature, ast.KindGetAccessor,
		ast.KindSetAccessor:
		facts.docSite(node, node.Name(), syntax.DocMethod, owner)
	case ast.KindConstructor:
		facts.docSiteNamed(node, "constructor", syntax.DocConstructor, owner)
	case ast.KindPropertyDeclaration, ast.KindPropertySignature, ast.KindEnumMember:
		facts.docSite(node, node.Name(), syntax.DocField, owner)
	}
}

func (facts *docFacts) docVariable(node *ast.Node, owner string) {
	declarations := node.AsVariableStatement().DeclarationList.AsVariableDeclarationList()
	if len(declarations.Declarations.Nodes) == 0 {
		return
	}
	first := declarations.Declarations.Nodes[0]
	kind := syntax.DocVariable
	if isFunctionValue(first.Initializer()) {
		kind = syntax.DocFunction
	}
	facts.docSite(node, first.Name(), kind, owner)
}

func isFunctionValue(value *ast.Node) bool {
	for value != nil && isWrapper(value.Kind) {
		value = value.Expression()
	}
	return value != nil &&
		(value.Kind == ast.KindArrowFunction || value.Kind == ast.KindFunctionExpression)
}

func (facts *docFacts) docSite(node, name *ast.Node, kind syntax.DocKind, owner string) {
	facts.docSiteNamed(node, facts.nameOr(name, ""), kind, owner)
}

func (facts *docFacts) docSiteNamed(
	node *ast.Node, name string, kind syntax.DocKind, owner string,
) {
	doc, found := facts.docComment(node)
	if !found {
		return
	}
	facts.out = append(facts.out, syntax.DocSite{
		Symbol:         qualified(owner, name),
		Kind:           kind,
		Line:           facts.lines.Of(doc.start),
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

func (facts *docFacts) docComment(node *ast.Node) (docShape, bool) {
	ranges := slices.Collect(
		scanner.GetLeadingCommentRanges(facts.factory, facts.text, node.Pos()),
	)
	boundary := scanner.SkipTrivia(facts.text, node.Pos())
	first := len(ranges)
	for first > 0 && strings.Count(facts.text[ranges[first-1].End():boundary], "\n") <= 1 {
		first--
		boundary = ranges[first].Pos()
	}
	chain := ranges[first:]
	doc := slices.IndexFunc(chain, func(comment ast.CommentRange) bool {
		return jsdoc.IsBlock(facts.text[comment.Pos():comment.End()])
	})
	if doc < 0 {
		return docShape{}, false
	}
	for next := doc + 1; next < len(chain); next++ {
		if jsdoc.IsBlock(facts.text[chain[next].Pos():chain[next].End()]) {
			doc = next
		}
	}
	return facts.shapeOf(chain, doc), true
}

func (facts *docFacts) shapeOf(chain []ast.CommentRange, doc int) docShape {
	var others []string
	for idx, comment := range chain {
		if idx != doc {
			others = append(others, facts.text[comment.Pos():comment.End()])
		}
	}
	shape := jsdoc.Classify(facts.text[chain[doc].Pos():chain[doc].End()], others)
	return docShape{start: chain[doc].Pos(), exempt: shape.Exempt, mixed: shape.Mixed}
}
