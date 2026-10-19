package narration

import (
	"go/ast"
	"go/token"
	"strings"
)

func fileSegments(file *ast.File, fset *token.FileSet) []segment {
	var segments []segment
	for _, group := range file.Comments {
		for _, comment := range group.List {
			segments = append(
				segments,
				commentSegments(comment, fset.Position(comment.Slash).Line)...)
		}
	}
	return segments
}

func commentSegments(comment *ast.Comment, firstLine int) []segment {
	if body, ok := strings.CutPrefix(comment.Text, "//"); ok {
		return []segment{{line: firstLine, text: body}}
	}
	body := strings.TrimSuffix(strings.TrimPrefix(comment.Text, "/*"), "*/")
	var segments []segment
	for offset, text := range strings.Split(body, "\n") {
		segments = append(segments, segment{line: firstLine + offset, text: text})
	}
	return segments
}
