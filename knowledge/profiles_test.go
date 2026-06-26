package knowledge

import (
	"slices"
	"testing"
)

// TestApplyProfiles asserts a profile rewrites the targeted skill's composition
// and that unknown profiles are rejected.
func TestApplyProfiles(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err = idx.ApplyProfiles([]string{"monorepo"}); err != nil {
		t.Fatalf("ApplyProfiles: %v", err)
	}
	skill, ok := idx.Skill("code-placement")
	if !ok {
		t.Fatal("missing code-placement skill")
	}
	want := []string{"code-placement-monorepo"}
	if !slices.Equal(skill.ComposeRules, want) {
		t.Errorf("code-placement ComposeRules = %v, want %v", skill.ComposeRules, want)
	}
	// The Skills slice must be updated too, not just the by-id map.
	for _, s := range idx.Skills {
		if s.ID == "code-placement" && !slices.Equal(s.ComposeRules, want) {
			t.Errorf("Skills slice not updated: %v", s.ComposeRules)
		}
	}
}

// TestApplyProfiles_Unknown asserts an unknown profile id is an error.
func TestApplyProfiles_Unknown(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = idx.ApplyProfiles([]string{"does-not-exist"}); err == nil {
		t.Error("ApplyProfiles(unknown) = nil, want error")
	}
}
