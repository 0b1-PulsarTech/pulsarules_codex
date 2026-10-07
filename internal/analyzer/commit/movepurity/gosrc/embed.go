package gosrc

import (
	"fmt"
	"go/ast"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
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

func EmbedCovers(patterns []string, rel string) bool {
	segments := strings.Split(rel, "/")
	for _, directive := range patterns {
		pattern, isAll := strings.CutPrefix(directive, "all:")
		for depth := range segments {
			matched, err := path.Match(pattern, strings.Join(segments[:depth+1], "/"))
			if err != nil || !matched {
				continue
			}
			// A matched directory embeds its tree minus dot and underscore names, unless all:.
			if isAll || !slices.ContainsFunc(segments[depth+1:], gomod.IsHiddenName) {
				return true
			}
		}
	}
	return false
}

func EmbedFiles(
	patterns []string,
	root string,
	children func(dir string) ([]string, error),
) (map[string]bool, error) {
	files := map[string]bool{}
	prefix := root + "/"
	if root == "." {
		prefix = ""
	}
	var walk func(entry string) error
	walk = func(entry string) error {
		entries, err := children(entry)
		if err != nil {
			return fmt.Errorf("list %s: %w", entry, err)
		}
		rel := strings.TrimPrefix(entry, prefix)
		if len(entries) == 0 && entry != root && EmbedCovers(patterns, rel) {
			files[rel] = true
		}
		for _, child := range entries {
			if !embedReaches(patterns, strings.TrimPrefix(child, prefix)) {
				continue
			}
			if walkErr := walk(child); walkErr != nil {
				return walkErr
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	return files, nil
}

func embedReaches(patterns []string, rel string) bool {
	if EmbedCovers(patterns, rel) {
		return true
	}
	segments := strings.Split(rel, "/")
	for _, directive := range patterns {
		pattern, _ := strings.CutPrefix(directive, "all:")
		parts := strings.Split(pattern, "/")
		if len(parts) > len(segments) && prefixMatches(parts, segments) {
			return true
		}
	}
	return false
}

func prefixMatches(parts, segments []string) bool {
	for depth, segment := range segments {
		if matched, err := path.Match(parts[depth], segment); err != nil || !matched {
			return false
		}
	}
	return true
}
