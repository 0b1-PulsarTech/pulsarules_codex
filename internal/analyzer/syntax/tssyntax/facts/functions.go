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

func (functions *functionFacts) enter(node *ast.Node) bool {
	if !isFunction(node.Kind) {
		return true
	}
	body := node.Body()
	if body == nil {
		return !functions.isOverload(node)
	}
	functions.frames = append(functions.frames, functions.record(node, body))
	return true
}

func (functions *functionFacts) leave(node *ast.Node) {
	if isFunction(node.Kind) && node.Body() != nil {
		functions.frames = functions.frames[:len(functions.frames)-1]
	}
}

func (functions *functionFacts) record(node, body *ast.Node) int {
	who := functions.identify(node)
	container := functions.isTest && who.kind == syntax.FuncClosure && isTestCallback(node)
	nested := who.kind == syntax.FuncClosure && !who.bound && len(functions.frames) > 0
	fn := syntax.Function{
		Name:          who.name,
		Kind:          who.kind,
		Line:          functions.declarationLine(node),
		BodyStart:     functions.lineAt(body.Pos()),
		BodyEnd:       functions.lines.Of(max(body.End()-1, body.Pos())),
		Params:        countParams(node),
		Complexity:    1,
		TestContainer: container,
		Nested:        nested,
	}
	if body.Kind == ast.KindBlock && (!nested || container) {
		functions.addBlocks(body)
	}
	functions.out = append(functions.out, fn)
	return len(functions.out) - 1
}

func (functions *functionFacts) declarationLine(node *ast.Node) int {
	if name := node.Name(); name != nil && ast.HasDecorators(node) {
		return functions.lineAt(name.Pos())
	}
	return functions.lineAt(node.Pos())
}

func (functions *functionFacts) isOverload(node *ast.Node) bool {
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
	own := functions.nameOr(node.Name(), "")
	for _, next := range siblings[index+1:] {
		if next.Kind != node.Kind || functions.nameOr(next.Name(), "") != own {
			return false
		}
		if next.Body() != nil {
			return true
		}
	}
	return false
}
