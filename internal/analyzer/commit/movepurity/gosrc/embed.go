package gosrc

import (
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

func EmbedFiles(patterns []string, root string, files []string) map[string]bool {
	var rels []string
	for _, file := range files {
		rel, isUnder := file, true
		if root != "." {
			rel, isUnder = strings.CutPrefix(file, root+"/")
		}
		if isUnder {
			rels = append(rels, rel)
		}
	}
	modules := map[string]bool{}
	for _, rel := range rels {
		if path.Base(rel) == "go.mod" {
			modules[path.Dir(rel)] = true
		}
	}
	covered := map[string]bool{}
	for _, rel := range rels {
		if EmbedCovers(patterns, rel) && !inNestedModule(rel, modules) {
			covered[rel] = true
		}
	}
	return covered
}

func inNestedModule(rel string, modules map[string]bool) bool {
	for dir := path.Dir(rel); dir != "."; dir = path.Dir(dir) {
		if modules[dir] {
			return true
		}
	}
	return false
}
