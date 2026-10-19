package facts

import (
	"slices"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/ast"
)

type complexityFacts struct{ functions *functionFacts }

func (facts complexityFacts) visit(node *ast.Node) {
	if !branches(node) {
		return
	}
	for _, frame := range slices.Backward(facts.functions.frames) {
		fn := &facts.functions.out[frame]
		fn.Complexity++
		if !fn.Nested || fn.TestContainer {
			return
		}
	}
}

func branches(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindIfStatement, ast.KindForStatement, ast.KindForInStatement,
		ast.KindForOfStatement, ast.KindWhileStatement, ast.KindDoStatement,
		ast.KindCaseClause, ast.KindCatchClause, ast.KindConditionalExpression:
		return true
	case ast.KindBinaryExpression:
		return isShortCircuit(node.AsBinaryExpression().OperatorToken.Kind)
	}
	return false
}

func isShortCircuit(kind ast.Kind) bool {
	return kind == ast.KindAmpersandAmpersandToken || kind == ast.KindBarBarToken ||
		kind == ast.KindQuestionQuestionToken
}
