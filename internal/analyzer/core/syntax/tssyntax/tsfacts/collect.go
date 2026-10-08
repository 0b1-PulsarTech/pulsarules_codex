package tsfacts

import (
	"errors"
	"slices"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/scanner"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsparse"
)

const MaxTreeDepth = 50_000

var ErrTooDeep = errors.New("syntax tree is nested too deeply")

type collector struct {
	text   string
	lines  tsparse.Lines
	isTest bool
	out    syntax.File
	frames []int
	locals map[string]bool
	depth  int
	stop   bool
	each   ast.Visitor
}

type Options struct{ IsTest bool }

func Collect(root *ast.SourceFile, text string, options Options) (syntax.File, error) {
	coll := &collector{
		text:   text,
		lines:  tsparse.NewLines(text),
		isTest: options.IsTest,
	}
	coll.each = func(child *ast.Node) bool {
		coll.walk(child)
		return coll.stop
	}
	node := root.AsNode()
	statements := node.Statements()
	coll.locals = exportedLocals(statements)
	node.ForEachChild(coll.each)
	if coll.stop {
		return syntax.File{}, ErrTooDeep
	}
	return coll.out, nil
}

func (coll *collector) walk(node *ast.Node) {
	// The parser appends cloned CommonJS export nodes to a JS file's statements.
	if node.Flags&ast.NodeFlagsReparsed != 0 {
		return
	}
	if coll.depth >= MaxTreeDepth {
		coll.stop = true
		return
	}
	coll.depth++
	coll.visit(node)
	coll.depth--
}

func (coll *collector) visit(node *ast.Node) {
	switch node.Kind {
	case ast.KindFunctionDeclaration, ast.KindFunctionExpression, ast.KindArrowFunction,
		ast.KindMethodDeclaration, ast.KindConstructor, ast.KindGetAccessor, ast.KindSetAccessor:
		coll.function(node)
	default:
		coll.complexity(node)
		coll.names(node)
		node.ForEachChild(coll.each)
	}
}

func (coll *collector) lineAt(offset int) int {
	return coll.lines.Of(scanner.SkipTrivia(coll.text, offset))
}

func (coll *collector) sourceOf(node *ast.Node) string {
	return coll.text[scanner.SkipTrivia(coll.text, node.Pos()):node.End()]
}

func (coll *collector) ownerName(constructor *ast.Node) string {
	owner := constructor.Parent
	if owner == nil {
		return "constructor"
	}
	if owner.Name() == nil && owner.Kind == ast.KindClassExpression {
		if bound := coll.closureIdentity(owner); bound.bound {
			return bound.name
		}
	}
	return coll.nameOr(owner.Name(), "default")
}

func (coll *collector) declarationLine(node *ast.Node) int {
	if name := node.Name(); name != nil && ast.HasDecorators(node) {
		return coll.lineAt(name.Pos())
	}
	return coll.lineAt(node.Pos())
}

func (coll *collector) isOverload(node *ast.Node) bool {
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
	own := coll.nameOr(node.Name(), "")
	for _, next := range siblings[index+1:] {
		if next.Kind != node.Kind || coll.nameOr(next.Name(), "") != own {
			return false
		}
		if next.Body() != nil {
			return true
		}
	}
	return false
}
