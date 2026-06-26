package render

import (
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
