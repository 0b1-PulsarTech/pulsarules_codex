package facts

import (
	"errors"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const MaxTreeDepth = 50_000

var errTooDeep = errors.New("syntax tree is nested too deeply")

type Options struct {
	Local  []string
	IsTest bool
}

type nodeVisitor interface{ visit(node *ast.Node) }

type walker struct {
	functions *functionFacts
	visitors  []nodeVisitor
	depth     int
	stop      bool
	each      ast.Visitor
}

func Collect(root *ast.SourceFile, text string, options Options) (syntax.File, error) {
	src := newSource(text)
	node := root.AsNode()
	statements := node.Statements()
	functions := &functionFacts{source: src, isTest: options.IsTest}
	names := &nameFacts{source: src, locals: exportedLocals(statements)}
	elses := &elseFacts{source: src}
	imports := &importFacts{source: src, local: options.Local}
	imports.collect(statements)
	docs := &docFacts{source: src, factory: ast.NewNodeFactory(ast.NodeFactoryHooks{})}
	docs.docStatements(statements, "")
	if err := newWalker(functions, complexityFacts{functions}, names, elses).run(node); err != nil {
		return syntax.File{}, err
	}
	file := syntax.File{
		Functions: functions.out,
		Blocks:    functions.blocks,
		Names:     names.out,
		Imports:   imports.out,
		DocSites:  docs.out,
		Elses:     elses.out,
	}
	if len(imports.out) > 0 {
		file.ImportGroups = importGroupNames()
	}
	return file, nil
}

func newWalker(functions *functionFacts, visitors ...nodeVisitor) *walker {
	walk := &walker{functions: functions, visitors: visitors}
	walk.each = func(child *ast.Node) bool {
		walk.walk(child)
		return walk.stop
	}
	return walk
}

func (walk *walker) run(root *ast.Node) error {
	root.ForEachChild(walk.each)
	if walk.stop {
		return errTooDeep
	}
	return nil
}

func (walk *walker) walk(node *ast.Node) {
	// The parser appends cloned CommonJS export nodes to a JS file's statements.
	if node.Flags&ast.NodeFlagsReparsed != 0 {
		return
	}
	if walk.depth >= MaxTreeDepth {
		walk.stop = true
		return
	}
	walk.depth++
	walk.visit(node)
	walk.depth--
}

func (walk *walker) visit(node *ast.Node) {
	if !walk.functions.enter(node) {
		return
	}
	for _, visitor := range walk.visitors {
		visitor.visit(node)
	}
	node.ForEachChild(walk.each)
	walk.functions.leave(node)
}
