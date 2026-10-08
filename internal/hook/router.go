package hook

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/slicesx"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Router struct {
	index *knowledge.Index
}

func NewRouter(index *knowledge.Index) *Router {
	return &Router{index: index}
}

func (router *Router) SkillsForFile(filePath string) []string {
	// A nil index routes nothing: a hook must never block a turn on a missing knowledge base.
	if router == nil || router.index == nil {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return nil
	}

	var ids []string
	if languages := fileLanguages(ext); len(languages) > 0 {
		ids = append(ids, router.index.BaselineFor(languages...)...)
	}
	ids = append(ids, router.triggered(filePath, ext)...)
	return slicesx.Dedupe(ids)
}

func (router *Router) Routes(filePath string) bool {
	if router == nil || router.index == nil {
		return false
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return false
	}
	// A Go file, or one a skill names by extension; the baseline alone never makes a
	// Dart or TS file noisy.
	if ext == ".go" && len(router.index.Router.Baseline.Always) > 0 {
		return true
	}
	return len(router.triggered(filePath, ext)) > 0
}

func (router *Router) triggered(filePath, ext string) []string {
	var ids []string
	for _, skill := range router.index.Skills {
		if triggersName(skill.Triggers, ext) {
			ids = append(ids, skill.ID)
		}
	}
	if strings.HasSuffix(strings.ToLower(filePath), "_test.go") {
		for _, id := range router.dispatchSkills("test") {
			// The word also matches the _test.dart row, whose skills a Go test must not list.
			if router.index.Fits(id, "go") {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func fileLanguages(ext string) []string {
	switch ext {
	case ".go":
		return []string{"go"}
	case ".dart":
		return []string{"dart"}
	case ".svelte":
		return []string{"svelte"}
	}
	return nil
}

func (router *Router) dispatchSkills(keyword string) []string {
	var ids []string
	for _, row := range router.index.Router.Dispatch {
		// A new row naming an existing keyword is picked up with no Go change.
		if hasWord(row.Signal, keyword) {
			ids = append(ids, row.Skills...)
		}
	}
	return ids
}

// simplification: could over-match ext as a prefix (".sql" in ".sqlite");
// acceptable while hand-authored. Upgrade path: exact suffix/word matching.
func triggersName(triggers []string, ext string) bool {
	for _, trigger := range triggers {
		if strings.Contains(strings.ToLower(trigger), ext) {
			return true
		}
	}
	return false
}

func hasWord(text, word string) bool {
	for _, token := range strings.FieldsFunc(text, func(char rune) bool {
		return !unicode.IsLetter(char) && !unicode.IsDigit(char)
	}) {
		// Whole-token match only, so "sql" never matches inside "sqlc".
		if strings.EqualFold(token, word) {
			return true
		}
	}
	return false
}
