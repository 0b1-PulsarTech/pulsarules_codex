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
	if ext == ".go" {
		for _, entry := range router.index.Router.Baseline.Always {
			ids = append(ids, entry.Skill)
		}
	}
	for _, skill := range router.index.Skills {
		if triggersName(skill.Triggers, ext) {
			ids = append(ids, skill.ID)
		}
	}
	if strings.HasSuffix(strings.ToLower(filePath), "_test.go") {
		ids = append(ids, router.dispatchSkills("test")...)
	}
	return slicesx.Dedupe(ids)
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
