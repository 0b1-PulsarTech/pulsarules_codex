package directive

import (
	"go/ast"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func directivePrefixes() []string {
	return []string{
		"//go:", "// +build", "//nolint", "//lint:",
		"//goverter:", "// goverter:", "//pulsarules:", "// pulsarules:",
	}
}

func isBlankComment(text string) bool {
	body := strings.TrimPrefix(text, "//")
	body = strings.TrimSuffix(strings.TrimPrefix(body, "/*"), "*/")
	return strings.TrimSpace(body) == ""
}

func IsDirectiveLine(text string) bool {
	trimmed := strings.TrimSpace(text)
	if isBlankComment(trimmed) || core.IsExemptHeaderLine(trimmed) {
		return true
	}
	return slices.ContainsFunc(directivePrefixes(), func(prefix string) bool {
		return strings.HasPrefix(trimmed, prefix)
	})
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
	return group != nil && len(group.List) > 0 && core.IsSimplificationMarker(group.List[0].Text)
}

func proseLines(group *ast.CommentGroup) int {
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
	if hasDirective && proseLines(group) > 0 {
		return "delete the prose lines, keep the directive lines"
	}
	return fallback
}
