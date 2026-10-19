package render

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestBodyWithoutH1(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
		want  string
	}{
		{"strips h1 and blank line", "# Title\n\nbody\n", "body\n"},
		{"strips h1 only", "# Title\nbody\n", "body\n"},
		{"no h1", "just body\nmore\n", "more\n"},
		{"empty", "", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := bodyWithoutH1(testCase.input); got != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestComposeReferences_Deduplicated(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	goStyle, _ := idx.Skill("go-style")
	docs, err := composeReferences(idx, goStyle)
	if err != nil {
		t.Fatalf("composeReferences(go-style): %v", err)
	}
	if len(docs) == 0 || docs[0].Title != "Effective Go" {
		t.Fatalf("expected Effective Go first, got %+v", docs)
	}
	seen := map[string]bool{}
	for _, doc := range docs {
		if seen[doc.Title] {
			t.Errorf("reference %q listed twice", doc.Title)
		}
		seen[doc.Title] = true
	}
}

func TestEntryID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		entry string
		want  string
	}{
		{"a bare id", "commits", "commits"},
		{"an id with a section", "commits#header", "commits"},
		{"empty", "", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := entryID(testCase.entry); got != testCase.want {
				t.Errorf("entryID(%q) = %q, want %q", testCase.entry, got, testCase.want)
			}
		})
	}
}

func TestComposed(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	pattern := idx.Patterns[0]
	rule := idx.Rules[0]
	testCases := []struct {
		name     string
		rules    []string
		patterns []string
		stopAt   int
		want     []string
	}{
		{
			"rules before patterns, anchor stripped",
			[]string{rule.ID + "#must"},
			[]string{pattern.ID},
			0,
			[]string{"rules:" + rule.ID + ":true", "patterns:" + pattern.ID + ":true"},
		},
		{
			"dangling ids still yield, unfound",
			[]string{"no-such-rule"},
			[]string{"no-such-pattern"},
			0,
			[]string{"rules:no-such-rule:false", "patterns:no-such-pattern:false"},
		},
		{
			"stops when the consumer breaks",
			[]string{rule.ID, "no-such-rule"},
			[]string{pattern.ID},
			1,
			[]string{"rules:" + rule.ID + ":true"},
		},
		{"empty", nil, nil, 0, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var got []string
			for entry := range composed(idx, testCase.rules, testCase.patterns) {
				found := "false"
				if entry.Found {
					found = "true"
				}
				got = append(got, string(entry.Kind)+":"+entry.ID+":"+found)
				if testCase.stopAt > 0 && len(got) == testCase.stopAt {
					break
				}
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("composed = %v, want %v", got, testCase.want)
			}
		})
	}
}
