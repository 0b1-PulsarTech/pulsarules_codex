package rulesummary

import (
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func languageRules() map[string]map[syntax.Language]string {
	return map[string]map[syntax.Language]string{
		"import-groups": {syntax.LangDart: "dart-style"},
	}
}

func neutralFallbacks() map[string]string {
	return map[string]string{
		"doc-comment":        "comments",
		"package-file-count": "code-placement",
	}
}

func languageTags() map[string]bool {
	tags := map[string]bool{}
	for _, extension := range pass.DefaultSourceExtensions() {
		if language := syntax.LanguageForExtension(extension); language != "" {
			tags[string(language)] = true
		}
	}
	return tags
}

func goPinned(analyzerID string) bool {
	return analyzerID == "import-cycle" || strings.HasPrefix(analyzerID, "golangci-lint")
}

type resolver struct {
	index     *knowledge.Index
	defaults  map[string]string
	languages map[string]map[syntax.Language]string
	fallbacks map[string]string
	known     map[string]bool
}

func newResolver(index *knowledge.Index) resolver {
	return resolver{
		index:     index,
		defaults:  defaultRuleMap(),
		languages: languageRules(),
		fallbacks: neutralFallbacks(),
		known:     languageTags(),
	}
}

func (resolver resolver) resolve(analyzerID, file string) string {
	if file == "" || goPinned(analyzerID) {
		return lookupRule(resolver.defaults, analyzerID)
	}
	language := syntax.LanguageForExtension(filepath.Ext(file))
	candidate, ok := resolver.languages[analyzerID][language]
	if !ok {
		candidate = lookupRule(resolver.defaults, analyzerID)
	}
	if candidate == "" || resolver.fits(candidate, language) {
		return candidate
	}
	return resolver.fallbacks[analyzerID]
}

func (resolver resolver) fits(ruleID string, language syntax.Language) bool {
	rule, ok := resolver.index.Rule(ruleID)
	if !ok {
		return true
	}
	return rule.FitsLanguages(resolver.known, []string{string(language)})
}
