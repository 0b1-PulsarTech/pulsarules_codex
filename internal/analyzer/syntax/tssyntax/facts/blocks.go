package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func (functions *functionFacts) addBlocks(body *ast.Node) {
	functions.statements(body.Statements(), 0)
}

func (functions *functionFacts) statements(list []*ast.Node, depth int) {
	for _, stmt := range list {
		functions.statement(stmt, depth)
	}
}

func (functions *functionFacts) statement(node *ast.Node, depth int) {
	switch node.Kind {
	case ast.KindIfStatement:
		functions.ifStatement(node, depth)
	case ast.KindForStatement, ast.KindForInStatement, ast.KindForOfStatement,
		ast.KindWhileStatement, ast.KindDoStatement, ast.KindWithStatement:
		functions.child(node.Statement(), depth)
	case ast.KindLabeledStatement:
		functions.statement(node.Statement(), depth)
	case ast.KindSwitchStatement:
		functions.switchStatement(node, depth)
	case ast.KindTryStatement:
		functions.tryStatement(node, depth)
	case ast.KindBlock:
		functions.block(node, depth+1)
	}
}

func (functions *functionFacts) block(node *ast.Node, depth int) {
	functions.blocks = append(functions.blocks, syntax.Block{
		Line:  functions.lineAt(node.Pos()),
		Depth: depth,
	})
	functions.statements(node.Statements(), depth)
}

func (functions *functionFacts) child(node *ast.Node, depth int) {
	if node.Kind == ast.KindBlock {
		functions.block(node, depth+1)
		return
	}
	functions.statement(node, depth+1)
}

func (functions *functionFacts) ifStatement(node *ast.Node, depth int) {
	branch := node.AsIfStatement()
	functions.child(branch.ThenStatement, depth)
	other := branch.ElseStatement
	switch {
	case other == nil:
	case other.Kind == ast.KindIfStatement:
		functions.ifStatement(other, depth)
	default:
		functions.child(other, depth)
	}
}

func (functions *functionFacts) switchStatement(node *ast.Node, depth int) {
	caseBlock := node.AsSwitchStatement().CaseBlock
	functions.blocks = append(functions.blocks, syntax.Block{
		Line:  functions.lineAt(caseBlock.Pos()),
		Depth: depth + 1,
	})
	for _, clause := range caseBlock.AsCaseBlock().Clauses.Nodes {
		functions.statements(clause.Statements(), depth+1)
	}
}

func (functions *functionFacts) tryStatement(node *ast.Node, depth int) {
	attempt := node.AsTryStatement()
	functions.block(attempt.TryBlock, depth+1)
	if attempt.CatchClause != nil {
		functions.block(attempt.CatchClause.AsCatchClause().Block, depth+1)
	}
	if attempt.FinallyBlock != nil {
		functions.block(attempt.FinallyBlock, depth+1)
	}
}
