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

func languageSet(idx *Index) []string {
	var languages []string
	for _, skill := range idx.Skills {
		languages = append(languages, skill.Languages...)
	}
	return languages
}

func isRequired(ruleTags, composerLanguages, known []string) bool {
	if len(composerLanguages) == 0 {
		return true
	}
	tagged := false
	for _, tag := range ruleTags {
		if !slices.Contains(known, tag) {
			continue
		}
		tagged = true
		if slices.Contains(composerLanguages, tag) {
			return true
		}
	}
	return !tagged
}

func TestIsRequired(t *testing.T) {
	t.Parallel()

	known := []string{"go", "dart"}
	testCases := []struct {
		name      string
		ruleTags  []string
		languages []string
		want      bool
	}{
		{"go rule in dart skill", []string{"style", "go"}, []string{"dart"}, false},
		{"go rule in go skill", []string{"go"}, []string{"go"}, true},
		{"neutral rule in dart skill", []string{"style"}, []string{"dart"}, true},
		{"go rule in workflow", []string{"go"}, nil, true},
		{"dart rule in go skill", []string{"dart"}, []string{"go"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := isRequired(testCase.ruleTags, testCase.languages, known)
			if got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
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
			if !isRequired(testCase.rule.Tags, owner.languages, known) {
				return
			}
			if !slices.Contains(owner.ruleIDs, testCase.rule.ID) {
				t.Fatalf("%s does not compose %s", testCase.composerID, testCase.rule.ID)
			}
		})
	}
}
