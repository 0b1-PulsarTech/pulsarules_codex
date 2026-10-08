package tsfacts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
)

func (coll *collector) blocks(body *ast.Node) {
	coll.statements(body.Statements(), 0)
}

func (coll *collector) statements(list []*ast.Node, depth int) {
	for _, stmt := range list {
		coll.statement(stmt, depth)
	}
}

func (coll *collector) statement(node *ast.Node, depth int) {
	switch node.Kind {
	case ast.KindIfStatement:
		coll.ifStatement(node, depth)
	case ast.KindForStatement, ast.KindForInStatement, ast.KindForOfStatement,
		ast.KindWhileStatement, ast.KindDoStatement, ast.KindWithStatement:
		coll.child(node.Statement(), depth)
	case ast.KindLabeledStatement:
		coll.statement(node.Statement(), depth)
	case ast.KindSwitchStatement:
		coll.switchStatement(node, depth)
	case ast.KindTryStatement:
		coll.tryStatement(node, depth)
	case ast.KindBlock:
		coll.block(node, depth+1)
	}
}

func (coll *collector) block(node *ast.Node, depth int) {
	coll.out.Blocks = append(coll.out.Blocks, syntax.Block{
		Line:  coll.lineAt(node.Pos()),
		Depth: depth,
	})
	coll.statements(node.Statements(), depth)
}

func (coll *collector) child(node *ast.Node, depth int) {
	if node.Kind == ast.KindBlock {
		coll.block(node, depth+1)
		return
	}
	coll.statement(node, depth+1)
}

func (coll *collector) ifStatement(node *ast.Node, depth int) {
	branch := node.AsIfStatement()
	coll.child(branch.ThenStatement, depth)
	other := branch.ElseStatement
	switch {
	case other == nil:
	case other.Kind == ast.KindIfStatement:
		coll.ifStatement(other, depth)
	default:
		coll.child(other, depth)
	}
}

func (coll *collector) switchStatement(node *ast.Node, depth int) {
	caseBlock := node.AsSwitchStatement().CaseBlock
	coll.out.Blocks = append(coll.out.Blocks, syntax.Block{
		Line:  coll.lineAt(caseBlock.Pos()),
		Depth: depth + 1,
	})
	for _, clause := range caseBlock.AsCaseBlock().Clauses.Nodes {
		coll.statements(clause.Statements(), depth+1)
	}
}

func (coll *collector) tryStatement(node *ast.Node, depth int) {
	attempt := node.AsTryStatement()
	coll.block(attempt.TryBlock, depth+1)
	if attempt.CatchClause != nil {
		coll.block(attempt.CatchClause.AsCatchClause().Block, depth+1)
	}
	if attempt.FinallyBlock != nil {
		coll.block(attempt.FinallyBlock, depth+1)
	}
}
