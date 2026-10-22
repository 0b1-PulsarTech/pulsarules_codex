package loopexit

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func findProcessLifetimeMarker(
	fset *token.FileSet, comments []*ast.CommentGroup, forLine int, codeLines map[int]bool,
) (string, bool) {
	for _, group := range comments {
		for _, comment := range group.List {
			line := fset.Position(comment.Pos()).Line
			if line != forLine && line != forLine-1 {
				continue
			}
			// A line-above marker counts only when that line carries no other code.
			if line == forLine-1 && codeLines[line] {
				continue
			}
			text := strings.TrimPrefix(comment.Text, "//")
			reason, ok := matchProcessLifetimeMarker(text)
			if !ok {
				continue
			}
			return reason, true
		}
	}
	return "", false
}

func matchProcessLifetimeMarker(text string) (string, bool) {
	marker, ok := lexer.PulsarulesMarker(text)
	if !ok {
		return "", false
	}
	reason, ok := strings.CutPrefix(marker, "process-lifetime")
	if !ok {
		return "", false
	}
	// Require a word boundary so "process-lifetimeX" is not a marker plus a reason.
	if reason != "" && reason[0] != ' ' && reason[0] != '\t' {
		return "", false
	}
	return strings.TrimSpace(reason), true
}

func linesWithCode(fset *token.FileSet, file *ast.File) map[int]bool {
	lines := make(map[int]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		lines[fset.Position(node.End()).Line] = true
		return true
	})
	return lines
}
