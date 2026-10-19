package facts

import (
	"slices"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

type functionFacts struct {
	source
	isTest bool
	frames []int
	out    []syntax.Function
	blocks []syntax.Block
}

func isFunction(kind ast.Kind) bool {
	switch kind {
	case ast.KindFunctionDeclaration, ast.KindFunctionExpression, ast.KindArrowFunction,
		ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor, ast.KindSetAccessor:
		return true
	}
	return false
}

func (facts *functionFacts) enter(node *ast.Node) bool {
	if !isFunction(node.Kind) {
		return true
	}
	body := node.Body()
	if body == nil {
		return !facts.isOverload(node)
	}
	facts.frames = append(facts.frames, facts.record(node, body))
	return true
}

func (facts *functionFacts) leave(node *ast.Node) {
	if isFunction(node.Kind) && node.Body() != nil {
		facts.frames = facts.frames[:len(facts.frames)-1]
	}
}

func (facts *functionFacts) record(node, body *ast.Node) int {
	who := facts.identify(node)
	container := facts.isTest && who.kind == syntax.FuncClosure && isTestCallback(node)
	nested := who.kind == syntax.FuncClosure && !who.bound && len(facts.frames) > 0
	fn := syntax.Function{
		Name:          who.name,
		Kind:          who.kind,
		Line:          facts.declarationLine(node),
		BodyStart:     facts.lineAt(body.Pos()),
		BodyEnd:       facts.lines.Of(max(body.End()-1, body.Pos())),
		Params:        countParams(node),
		Complexity:    1,
		TestContainer: container,
		Nested:        nested,
	}
	if body.Kind == ast.KindBlock && (!nested || container) {
		facts.addBlocks(body)
	}
	facts.out = append(facts.out, fn)
	return len(facts.out) - 1
}

func (facts *functionFacts) declarationLine(node *ast.Node) int {
	if name := node.Name(); name != nil && ast.HasDecorators(node) {
		return facts.lineAt(name.Pos())
	}
	return facts.lineAt(node.Pos())
}

func (facts *functionFacts) isOverload(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindFunctionDeclaration, ast.KindMethodDeclaration, ast.KindConstructor:
	default:
		return false
	}
	if node.Parent == nil {
		return false
	}
	var siblings []*ast.Node
	if node.Parent.CanHaveStatements() {
		siblings = node.Parent.Statements()
	} else {
		siblings = node.Parent.Members()
	}
	index := slices.Index(siblings, node)
	if index < 0 {
		return false
	}
	own := facts.nameOr(node.Name(), "")
	for _, next := range siblings[index+1:] {
		if next.Kind != node.Kind || facts.nameOr(next.Name(), "") != own {
			return false
		}
		if next.Body() != nil {
			return true
		}
	}
	return false
}
