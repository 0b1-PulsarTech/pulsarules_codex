package render

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestBodyWithoutH1 asserts the leading `# Heading` line is dropped (with its
// trailing blank lines) so a transcluded body nests cleanly, while bodies with
// no H1 are returned unchanged.
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

// TestComposeReferences asserts a skill gathers the references cited by its
// composed rules (resolved, de-duplicated).
func TestComposeReferences(t *testing.T) {
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
