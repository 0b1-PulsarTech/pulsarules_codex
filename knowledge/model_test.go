package knowledge

import (
	"slices"
	"testing"
)

func TestRuleFitsLanguages(t *testing.T) {
	t.Parallel()

	known := map[string]bool{"go": true, "dart": true, "typescript": true, "javascript": true}
	bothWeb := []string{"typescript", "javascript"}
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
		{"ts tag on js", []string{"style", "typescript"}, []string{"javascript"}, false},
		{"two tags one matches", bothWeb, []string{"javascript"}, true},
		{"two tags none match", bothWeb, []string{"go"}, false},
		{"tag on no language", []string{"typescript"}, []string{""}, false},
		{"no tags", nil, []string{"typescript"}, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rule := Rule{Tags: testCase.ruleTags}
			if got := rule.FitsLanguages(known, testCase.languages); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestSkillOverride_Apply(t *testing.T) {
	t.Parallel()

	base := Skill{ComposeRules: []string{"r1"}, ComposePatterns: []string{"p1"}}
	testCases := []struct {
		name         string
		override     SkillOverride
		wantRules    []string
		wantPatterns []string
	}{
		{"nil override keeps both", SkillOverride{}, []string{"r1"}, []string{"p1"}},
		{
			"rules only",
			SkillOverride{ComposeRules: []string{"r2"}},
			[]string{"r2"},
			[]string{"p1"},
		},
		{
			"patterns only",
			SkillOverride{ComposePatterns: []string{"p2"}},
			[]string{"r1"},
			[]string{"p2"},
		},
		{
			"empty non-nil replaces both",
			SkillOverride{ComposeRules: []string{}, ComposePatterns: []string{}},
			[]string{},
			[]string{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			skill := base
			testCase.override.Apply(&skill)
			if !slices.Equal(skill.ComposeRules, testCase.wantRules) {
				t.Errorf("ComposeRules = %v, want %v", skill.ComposeRules, testCase.wantRules)
			}
			if !slices.Equal(skill.ComposePatterns, testCase.wantPatterns) {
				t.Errorf("ComposePatterns = %v, want %v",
					skill.ComposePatterns, testCase.wantPatterns)
			}
		})
	}
}
