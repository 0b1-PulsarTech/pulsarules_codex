package knowledge

import (
	"slices"
	"testing"
)

type composer struct {
	ruleIDs   []string
	languages []string
}

func contentRules(idx *Index) []Rule {
	contentAnalyzers := []string{"file-size", "text-markers", "typographic-markers"}
	var rules []Rule
	for _, rule := range idx.Rules {
		for _, analyzerID := range contentAnalyzers {
			if slices.Contains(rule.Analyzers, analyzerID) {
				rules = append(rules, rule)
				break
			}
		}
	}
	return rules
}

func composers(idx *Index) map[string]composer {
	found := map[string]composer{}
	for _, skill := range idx.Skills {
		found["skill "+skill.ID] = composer{skill.ComposeRules, skill.Languages}
	}
	for _, workflow := range idx.Workflows {
		found["workflow "+workflow.ID] = composer{ruleIDs: workflow.ComposesRules}
	}
	return found
}

func languageSet(idx *Index) map[string]bool {
	languages := map[string]bool{}
	for _, skill := range idx.Skills {
		for _, language := range skill.Languages {
			languages[language] = true
		}
	}
	return languages
}

func TestContentRulesAreComposed(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rules := contentRules(idx)
	if len(rules) == 0 {
		t.Fatal("no rule lists a content analyzer")
	}
	found := composers(idx)
	known := languageSet(idx)

	type composeCase struct {
		name, composerID string
		rule             Rule
	}
	composerIDs := []string{"skill go-style", "skill dart-style", "workflow refactoring"}
	testCases := make([]composeCase, 0, len(rules)*len(composerIDs))
	for _, rule := range rules {
		for _, composerID := range composerIDs {
			name := rule.ID + " in " + composerID
			testCases = append(testCases, composeCase{name, composerID, rule})
		}
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			owner := found[testCase.composerID]
			if !testCase.rule.FitsLanguages(known, owner.languages) {
				return
			}
			if !slices.Contains(owner.ruleIDs, testCase.rule.ID) {
				t.Fatalf("%s does not compose %s", testCase.composerID, testCase.rule.ID)
			}
		})
	}
}
