package testfile

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestLanguageRulesCoverEveryLanguage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		rule rule
	}{
		{"go", languageRules()[syntax.LangGo]},
		{"dart", languageRules()[syntax.LangDart]},
		{"typescript", languageRules()[syntax.LangTypeScript]},
		{"javascript", languageRules()[syntax.LangJavaScript]},
		{"svelte", languageRules()[syntax.LangSvelte]},
		{"fallback", fallbackRule()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if len(testCase.rule.markers)+len(testCase.rule.suffixes) == 0 {
				t.Errorf("rule %q has no markers or suffixes", testCase.name)
			}
		})
	}
}
