package facts

import (
	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func (facts *functionFacts) addBlocks(body *ast.Node) {
	facts.statements(body.Statements(), 0)
}

func (facts *functionFacts) statements(list []*ast.Node, depth int) {
	for _, stmt := range list {
		facts.statement(stmt, depth)
	}
}

func (facts *functionFacts) statement(node *ast.Node, depth int) {
	switch node.Kind {
	case ast.KindIfStatement:
		facts.ifStatement(node, depth)
	case ast.KindForStatement, ast.KindForInStatement, ast.KindForOfStatement,
		ast.KindWhileStatement, ast.KindDoStatement, ast.KindWithStatement:
		facts.child(node.Statement(), depth)
	case ast.KindLabeledStatement:
		facts.statement(node.Statement(), depth)
	case ast.KindSwitchStatement:
		facts.switchStatement(node, depth)
	case ast.KindTryStatement:
		facts.tryStatement(node, depth)
	case ast.KindBlock:
		facts.block(node, depth+1)
	}
}

func (facts *functionFacts) block(node *ast.Node, depth int) {
	facts.blocks = append(facts.blocks, syntax.Block{
		Line:  facts.lineAt(node.Pos()),
		Depth: depth,
	})
	facts.statements(node.Statements(), depth)
}

func (facts *functionFacts) child(node *ast.Node, depth int) {
	if node.Kind == ast.KindBlock {
		facts.block(node, depth+1)
		return
	}
	facts.statement(node, depth+1)
}

func (facts *functionFacts) ifStatement(node *ast.Node, depth int) {
	branch := node.AsIfStatement()
	facts.child(branch.ThenStatement, depth)
	other := branch.ElseStatement
	switch {
	case other == nil:
	case other.Kind == ast.KindIfStatement:
		facts.ifStatement(other, depth)
	default:
		facts.child(other, depth)
	}
}

func (facts *functionFacts) switchStatement(node *ast.Node, depth int) {
	caseBlock := node.AsSwitchStatement().CaseBlock
	facts.blocks = append(facts.blocks, syntax.Block{
		Line:  facts.lineAt(caseBlock.Pos()),
		Depth: depth + 1,
	})
	for _, clause := range caseBlock.AsCaseBlock().Clauses.Nodes {
		facts.statements(clause.Statements(), depth+1)
	}
}

func (facts *functionFacts) tryStatement(node *ast.Node, depth int) {
	attempt := node.AsTryStatement()
	facts.block(attempt.TryBlock, depth+1)
	if attempt.CatchClause != nil {
		facts.block(attempt.CatchClause.AsCatchClause().Block, depth+1)
	}
	if attempt.FinallyBlock != nil {
		facts.block(attempt.FinallyBlock, depth+1)
	}
}
