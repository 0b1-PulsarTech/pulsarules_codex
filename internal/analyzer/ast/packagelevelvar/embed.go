package packagelevelvar

import (
	"go/ast"
	"go/token"
	"strings"
)

func embedGroups(file *ast.File) []*ast.CommentGroup {
	var groups []*ast.CommentGroup
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:embed") {
				groups = append(groups, group)
				break
			}
		}
	}
	return groups
}

func (scan fileScan) hasEmbed(prevEnd, specPos token.Pos) bool {
	prevLine := 0
	if prevEnd.IsValid() {
		prevLine = scan.fset.Position(prevEnd).Line
	}
	// Only blank lines and comments may sit between a directive and its spec.
	for _, group := range scan.embeds {
		if group.End() <= specPos && scan.fset.Position(group.Pos()).Line > prevLine {
			return true
		}
	}
	return false
}
