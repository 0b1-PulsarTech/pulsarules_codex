package hook

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// Router resolves which skills a hook event should announce from the
// knowledge index's own data - router.yaml's always-load baseline and
// dispatch table, plus each skill's own triggers - instead of a hardcoded
// per-extension map. Adding a skill to skills.yaml with a trigger naming the
// file extension or activity it covers makes the hook route it with no Go
// change.
type Router struct {
	index *knowledge.Index
}

// NewRouter builds a Router over index. A nil index makes every lookup
// degrade to "no skills routed" rather than panicking, since a hook must
// never block a turn on a missing knowledge base.
func NewRouter(index *knowledge.Index) *Router {
	return &Router{index: index}
}

// SkillsForFile returns the skill IDs relevant to editing filePath: the
// router.yaml always-load baseline for any .go file ("every Go change"), any
// skill whose trigger literally names the file's extension (e.g.
// database-persistence's ".sql" trigger, grpc-adapter's ".proto" trigger),
// and the "Any test work" dispatch row's skills when filePath is a Go test
// file. It returns nil for a nil Router/index or an extension with no match.
func (r *Router) SkillsForFile(filePath string) []string {
	if r == nil || r.index == nil {
		return nil
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext == "" {
		return nil
	}

	var ids []string
	if ext == ".go" {
		for _, entry := range r.index.Router.Baseline.Always {
			ids = append(ids, entry.Skill)
		}
	}
	for _, skill := range r.index.Skills {
		if triggersName(skill.Triggers, ext) {
			ids = append(ids, skill.ID)
		}
	}
	if strings.HasSuffix(strings.ToLower(filePath), "_test.go") {
		ids = append(ids, r.dispatchSkills("test")...)
	}
	return dedupe(ids)
}

// dispatchSkills returns the union of skills from every router.yaml dispatch
// row whose signal names keyword as a whole word (e.g. "test" matches the
// "Any test work" row), so a new dispatch row naming an existing keyword is
// picked up without a Go change.
func (r *Router) dispatchSkills(keyword string) []string {
	var ids []string
	for _, row := range r.index.Router.Dispatch {
		if hasWord(row.Signal, keyword) {
			ids = append(ids, row.Skills...)
		}
	}
	return ids
}

// triggersName reports whether any trigger literally mentions ext (e.g.
// ".sql"), matched as a case-insensitive substring.
//
// simplification: a substring match could over-match a longer extension that
// contains ext as a prefix (".sql" inside a hypothetical ".sqlite" trigger);
// the ceiling is acceptable while triggers stay hand-authored prose. A skill
// needing precise disambiguation should phrase its trigger to avoid it.
func triggersName(triggers []string, ext string) bool {
	for _, trigger := range triggers {
		if strings.Contains(strings.ToLower(trigger), ext) {
			return true
		}
	}
	return false
}

// hasWord reports whether text contains word as a whole word,
// case-insensitively - never matching a substring inside a longer word (so
// "sql" never matches "sqlc").
func hasWord(text, word string) bool {
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if strings.EqualFold(token, word) {
			return true
		}
	}
	return false
}

// dedupe returns ids with duplicates removed, preserving first-seen order.
func dedupe(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
