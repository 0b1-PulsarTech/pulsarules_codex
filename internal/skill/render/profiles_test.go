package render

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestRenderWithProfile_SelectedVariant(t *testing.T) {
	t.Parallel()

	const (
		monorepoHeading = "Code placement - monorepo layout"
		innerHeading    = "Code placement - single repo with inner modules"
	)

	testCases := []struct {
		name     string
		layout   string
		wantBody string
		absent   string
	}{
		{"baseline has neither variant", "", "", monorepoHeading},
		{"monorepo composes monorepo variant", "monorepo", monorepoHeading, innerHeading},
		{"inner-modules composes inner variant", "inner-modules", innerHeading, monorepoHeading},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			body := renderCodePlacement(t, testCase.layout)
			if testCase.wantBody != "" && !strings.Contains(body, testCase.wantBody) {
				t.Errorf("layout %q: missing %q", testCase.layout, testCase.wantBody)
			}
			if strings.Contains(body, testCase.absent) {
				t.Errorf("layout %q: unexpectedly contains %q", testCase.layout, testCase.absent)
			}
		})
	}
}

func renderCodePlacement(t *testing.T, layout string) string {
	t.Helper()
	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if layout != "" {
		if err = idx.ApplyProfiles([]string{layout}); err != nil {
			t.Fatalf("ApplyProfiles(%q): %v", layout, err)
		}
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	skill, ok := idx.Skill("code-placement")
	if !ok {
		t.Fatal("missing code-placement skill")
	}
	body, err := rnd.RenderSkill(idx, skill, nil)
	if err != nil {
		t.Fatalf("RenderSkill: %v", err)
	}
	return body
}
