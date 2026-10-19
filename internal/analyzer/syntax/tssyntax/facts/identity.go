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

func (src source) identify(node *ast.Node) identity {
	switch node.Kind {
	case ast.KindFunctionDeclaration:
		return identity{name: src.nameOr(node.Name(), "default"), kind: syntax.FuncFunction}
	case ast.KindMethodDeclaration:
		return identity{name: src.nameOr(node.Name(), ""), kind: syntax.FuncMethod}
	case ast.KindConstructor:
		return identity{name: src.ownerName(node), kind: syntax.FuncConstructor}
	case ast.KindGetAccessor, ast.KindSetAccessor:
		return identity{name: src.nameOr(node.Name(), ""), kind: syntax.FuncAccessor}
	}
	return src.closureIdentity(node)
}

func (src source) closureIdentity(node *ast.Node) identity {
	if own := node.Name(); own != nil {
		return identity{name: src.nameOr(own, ""), kind: syntax.FuncClosure, bound: true}
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
				name:  src.nameOr(owner.Name(), anonymousClosure),
				kind:  syntax.FuncClosure,
				bound: true,
			}
		case ast.KindExportAssignment:
			return identity{name: "default", kind: syntax.FuncClosure, bound: true}
		case ast.KindBinaryExpression:
			return src.assignedIdentity(owner)
		}
	}
	return identity{name: anonymousClosure, kind: syntax.FuncClosure}
}

func (src source) assignedIdentity(assignment *ast.Node) identity {
	assign := assignment.AsBinaryExpression()
	target := assign.Left
	if assign.OperatorToken.Kind != ast.KindEqualsToken {
		return identity{name: anonymousClosure, kind: syntax.FuncClosure}
	}
	if target.Kind == ast.KindPropertyAccessExpression {
		target = target.Name()
	}
	return identity{
		name:  src.nameOr(target, anonymousClosure),
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

func (src source) ownerName(constructor *ast.Node) string {
	owner := constructor.Parent
	if owner == nil {
		return "constructor"
	}
	if owner.Name() == nil && owner.Kind == ast.KindClassExpression {
		if bound := src.closureIdentity(owner); bound.bound {
			return bound.name
		}
	}
	return src.nameOr(owner.Name(), "default")
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

func isTestCallback(node *ast.Node) bool {
	call := node.Parent
	if call == nil || call.Kind != ast.KindCallExpression {
		return false
	}
	isArgument := false
	for _, argument := range call.Arguments() {
		isArgument = isArgument || argument == node
	}
	return isArgument && isTestCallee(call.AsCallExpression().Expression)
}

func isTestCallee(callee *ast.Node) bool {
	if callee.Kind == ast.KindPropertyAccessExpression {
		callee = callee.Expression()
	}
	if callee.Kind != ast.KindIdentifier {
		return false
	}
	switch callee.Text() {
	case "describe", "it", "test", "suite", "specify", "beforeEach", "afterEach",
		"beforeAll", "afterAll", "before", "after":
		return true
	}
	return false
}
