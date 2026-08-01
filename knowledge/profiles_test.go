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
	skill, ok := idx.Skill("code-placement")
	if !ok {
		t.Fatal("missing code-placement skill")
	}
	want := []string{"code-placement-monorepo"}
	if !slices.Equal(skill.ComposeRules, want) {
		t.Errorf("code-placement ComposeRules = %v, want %v", skill.ComposeRules, want)
	}
	for _, s := range idx.Skills {
		if s.ID == "code-placement" && !slices.Equal(s.ComposeRules, want) {
			t.Errorf("Skills slice not updated: %v", s.ComposeRules)
		}
	}
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
