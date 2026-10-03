package gosrc

import (
	"go/ast"
	"strconv"
	"strings"
	"unicode"
)

func embedPatterns(file *ast.File) []string {
	var patterns []string
	for _, group := range file.Comments {
		for _, comment := range group.List {
			args, found := strings.CutPrefix(comment.Text, "//go:embed")
			if found && (args == "" || unicode.IsSpace(rune(args[0]))) {
				patterns = append(patterns, splitEmbedArgs(args)...)
			}
		}
	}
	return patterns
}

func splitEmbedArgs(args string) []string {
	var patterns []string
	rest := strings.TrimSpace(args)
	for rest != "" {
		if quoted, err := strconv.QuotedPrefix(rest); err == nil {
			pattern, _ := strconv.Unquote(quoted)
			patterns = append(patterns, pattern)
			rest = strings.TrimSpace(rest[len(quoted):])
			continue
		}
		end := strings.IndexFunc(rest, unicode.IsSpace)
		if end < 0 {
			end = len(rest)
		}
		patterns = append(patterns, rest[:end])
		rest = strings.TrimSpace(rest[end:])
	}
	return patterns
}
