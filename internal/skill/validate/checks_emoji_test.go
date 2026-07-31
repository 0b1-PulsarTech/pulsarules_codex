package validate

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestEmojiThemeShortcodes asserts the embedded theme table, whose shortcodes
// really do live in the catalog, validates cleanly and does not depend on the
// knowledge index passed in.
func TestEmojiThemeShortcodes(t *testing.T) {
	t.Parallel()

	if problems := emojiThemeShortcodes(&knowledge.Index{}); len(problems) != 0 {
		t.Errorf("embedded themes should all resolve, got %v", problems)
	}
}

// TestThemeShortcodesInCatalog covers both the passing and failing path: a
// theme naming only catalog shortcodes is silent, one naming a shortcode
// absent from the catalog is reported by area and shortcode.
func TestThemeShortcodesInCatalog(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		anchors     []emoji.Anchor
		wantProblem string
	}{
		{
			name:        "all shortcodes in catalog",
			anchors:     []emoji.Anchor{{Area: "fix", Shortcodes: []string{"bug", "wrench"}}},
			wantProblem: "",
		},
		{
			name: "off-catalog shortcode",
			anchors: []emoji.Anchor{
				{Area: "move", Shortcodes: []string{"not_a_real_shortcode"}},
			},
			wantProblem: `emoji theme "move" lists off-catalog shortcode "not_a_real_shortcode"`,
		},
	}

	catalog, err := emoji.NewCatalog()
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			problems := themeShortcodesInCatalog(catalog, testCase.anchors)
			if testCase.wantProblem == "" {
				if len(problems) != 0 {
					t.Fatalf("expected no problems, got %v", problems)
				}
				return
			}
			if len(problems) != 1 || !strings.Contains(problems[0], testCase.wantProblem) {
				t.Fatalf("problems = %v, want one containing %q", problems, testCase.wantProblem)
			}
		})
	}
}
