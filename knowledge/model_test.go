package knowledge

import "testing"

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
