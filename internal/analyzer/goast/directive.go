package goast

import (
	"go/ast"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func isBlankComment(text string) bool {
	return lexer.CommentBody(text, lexer.DialectGo.Syntax()) == ""
}

func IsDirectiveLine(text string) bool {
	return isBlankComment(text) || lexer.IsDirectiveLine(lexer.DialectGo, text)
}

func IsDirectiveGroup(group *ast.CommentGroup) bool {
	if group == nil || len(group.List) == 0 {
		return true
	}
	return !slices.ContainsFunc(group.List, func(comment *ast.Comment) bool {
		return !IsDirectiveLine(comment.Text)
	})
}

func IsSimplificationGroup(group *ast.CommentGroup) bool {
	return group != nil && len(group.List) > 0 &&
		lexer.IsSimplificationMarker(group.List[0].Text, lexer.AnyCommentSyntax())
}

func ProseLines(group *ast.CommentGroup) int {
	if group == nil {
		return 0
	}
	total := 0
	for _, comment := range group.List {
		if !IsDirectiveLine(comment.Text) {
			total += strings.Count(comment.Text, "\n") + 1
		}
	}
	return total
}

func HintFor(group *ast.CommentGroup, fallback string) string {
	if group == nil {
		return fallback
	}
	hasDirective := slices.ContainsFunc(group.List, func(comment *ast.Comment) bool {
		return IsDirectiveLine(comment.Text) && !isBlankComment(comment.Text)
	})
	if hasDirective && ProseLines(group) > 0 {
		return "delete the prose lines, keep the directive lines"
	}
	return fallback
}
