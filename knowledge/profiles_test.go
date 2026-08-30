package knowledge

import (
	"slices"
	"testing"
)

func TestApplyProfiles_TargetedSkill(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err = idx.ApplyProfiles([]string{"monorepo"}); err != nil {
		t.Fatalf("ApplyProfiles: %v", err)
	}
	want := []string{"code-placement-monorepo"}

	t.Run("updates the by-id skill map", func(t *testing.T) {
		t.Parallel()
		skill, ok := idx.Skill("code-placement")
		if !ok {
			t.Fatal("missing code-placement skill")
		}
		if !slices.Equal(skill.ComposeRules, want) {
			t.Errorf("code-placement ComposeRules = %v, want %v", skill.ComposeRules, want)
		}
	})

	t.Run("also updates the Skills slice, not just the by-id map", func(t *testing.T) {
		t.Parallel()
		for _, skill := range idx.Skills {
			if skill.ID == "code-placement" && !slices.Equal(skill.ComposeRules, want) {
				t.Errorf("Skills slice not updated: %v", skill.ComposeRules)
			}
		}
	})
}

func TestApplyProfiles_UnknownProfileIDIsError(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = idx.ApplyProfiles([]string{"does-not-exist"}); err == nil {
		t.Error("ApplyProfiles(unknown) = nil, want error")
	}
}
