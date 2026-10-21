package rulesummary

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntaxreport"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func languageRules() map[string]map[language.ID]string {
	return map[string]map[language.ID]string{
		"import-groups": {language.Dart: "dart-style"},
	}
}

func neutralFallbacks() map[string]string {
	return map[string]string{
		"doc-comment":        "comments",
		"package-file-count": "code-placement",
	}
}

func languageTags() map[string]bool {
	// The languages a syntax provider parses; a rule tag outside the set is a topic.
	ids := syntaxreport.Languages()
	tags := make(map[string]bool, len(ids))
	for _, id := range ids {
		tags[string(id)] = true
	}
	return tags
}

func goPinned(analyzerID string) bool {
	return analyzerID == "import-cycle" || strings.HasPrefix(analyzerID, "golangci-lint")
}

type resolver struct {
	index     *knowledge.Index
	defaults  map[string]string
	languages map[string]map[language.ID]string
	fallbacks map[string]string
	known     map[string]bool
}

func newResolver(index *knowledge.Index, defaults map[string]string) resolver {
	return resolver{
		index:     index,
		defaults:  defaults,
		languages: languageRules(),
		fallbacks: neutralFallbacks(),
		known:     languageTags(),
	}
}

func (resolver resolver) resolve(analyzerID, file string) string {
	if file == "" || goPinned(analyzerID) {
		return lookupRule(resolver.defaults, analyzerID)
	}
	lang := language.ForExtension(filepath.Ext(file)).ID
	candidate, ok := resolver.languages[analyzerID][lang]
	if !ok {
		candidate = lookupRule(resolver.defaults, analyzerID)
	}
	if candidate == "" || resolver.fits(candidate, lang) {
		return candidate
	}
	return resolver.fallbacks[analyzerID]
}

func (resolver resolver) fits(ruleID string, lang language.ID) bool {
	rule, ok := resolver.index.Rule(ruleID)
	if !ok {
		return true
	}
	return rule.FitsLanguages(resolver.known, []string{string(lang)})
}
