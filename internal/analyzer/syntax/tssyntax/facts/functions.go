package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const anonymousClosure = "closure"

type identity struct {
	name  string
	kind  syntax.FunctionKind
	bound bool
}

func (coll *collector) function(node *ast.Node) {
	body := node.Body()
	if body == nil && coll.isOverload(node) {
		return
	}
	if body == nil {
		node.ForEachChild(coll.each)
		return
	}
	coll.frames = append(coll.frames, coll.enter(node, body))
	node.ForEachChild(coll.each)
	coll.frames = coll.frames[:len(coll.frames)-1]
}

func (coll *collector) enter(node, body *ast.Node) int {
	who := coll.identify(node)
	container := coll.isTest && who.kind == syntax.FuncClosure && isTestCallback(node)
	nested := who.kind == syntax.FuncClosure && !who.bound && len(coll.frames) > 0
	fn := syntax.Function{
		Name:          who.name,
		Kind:          who.kind,
		Line:          coll.declarationLine(node),
		BodyStart:     coll.lineAt(body.Pos()),
		BodyEnd:       coll.lines.Of(max(body.End()-1, body.Pos())),
		Params:        countParams(node),
		Complexity:    1,
		TestContainer: container,
		Nested:        nested,
	}
	if body.Kind == ast.KindBlock && (!nested || container) {
		coll.blocks(body)
	}
	coll.out.Functions = append(coll.out.Functions, fn)
	return len(coll.out.Functions) - 1
}

func (coll *collector) identify(node *ast.Node) identity {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		return identity{name: coll.nameOr(node.Name(), "default"), kind: syntax.FuncFunction}
	case ast.KindMethodDeclaration:
		return identity{name: coll.nameOr(node.Name(), ""), kind: syntax.FuncMethod}
	case ast.KindConstructor:
		return identity{name: coll.ownerName(node), kind: syntax.FuncConstructor}
	case ast.KindGetAccessor, ast.KindSetAccessor:
		return identity{name: coll.nameOr(node.Name(), ""), kind: syntax.FuncAccessor}
	}
	return coll.closureIdentity(node)
}

func (coll *collector) closureIdentity(node *ast.Node) identity {
	if own := node.Name(); own != nil {
		return identity{name: coll.nameOr(own, ""), kind: syntax.FuncClosure, bound: true}
	}
	owner := node.Parent
	for owner != nil && isWrapper(owner.Kind) {
		owner = owner.Parent
	}
	if owner != nil {
		switch owner.Kind {
		case ast.KindVariableDeclaration, ast.KindPropertyDeclaration,
			ast.KindPropertyAssignment:
			return identity{
				name:  coll.nameOr(owner.Name(), anonymousClosure),
				kind:  syntax.FuncClosure,
				bound: true,
			}
		case ast.KindExportAssignment:
			return identity{name: "default", kind: syntax.FuncClosure, bound: true}
		case ast.KindBinaryExpression:
			return coll.assignedIdentity(owner)
		}
	}
	return identity{name: anonymousClosure, kind: syntax.FuncClosure}
}

func (coll *collector) assignedIdentity(assignment *ast.Node) identity {
	assign := assignment.AsBinaryExpression()
	target := assign.Left
	if assign.OperatorToken.Kind != ast.KindEqualsToken {
		return identity{name: anonymousClosure, kind: syntax.FuncClosure}
	}
	if target.Kind == ast.KindPropertyAccessExpression {
		target = target.Name()
	}
	return identity{
		name:  coll.nameOr(target, anonymousClosure),
		kind:  syntax.FuncClosure,
		bound: true,
	}
}

func isWrapper(kind ast.Kind) bool {
	switch kind {
	case ast.KindParenthesizedExpression, ast.KindAsExpression, ast.KindSatisfiesExpression,
		ast.KindNonNullExpression, ast.KindTypeAssertionExpression:
		return true
	}
	return false
}

func (coll *collector) nameOr(name *ast.Node, fallback string) string {
	if name == nil {
		return fallback
	}
	switch name.Kind {
	case ast.KindIdentifier, ast.KindPrivateIdentifier, ast.KindStringLiteral,
		ast.KindNumericLiteral, ast.KindNoSubstitutionTemplateLiteral:
		if text := name.Text(); text != "" {
			return text
		}
		return fallback
	}
	return coll.sourceOf(name)
}

func countParams(node *ast.Node) int {
	total := 0
	for _, param := range node.Parameters() {
		if !isThisParam(param) {
			total++
		}
	}
	return total
}

func isThisParam(param *ast.Node) bool {
	name := param.Name()
	return name != nil && name.Kind == ast.KindIdentifier && name.Text() == "this"
}
