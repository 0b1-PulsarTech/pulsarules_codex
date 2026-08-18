package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// skillNormativeSections reports every skill whose composed rules and patterns
// render no non-empty "must", "forbidden", or "validation" section. Such a
// skill states no obligation - it is documentation, not a governed contract -
// and pulsarules_cli validate would otherwise stay green while a real contract
// silently disappears (a skill can compose a pattern that defines only
// "recipe", which is descriptive, not normative). It checks both a skill's
// base composition AND every profile's override of that composition
// (knowledge/standards/profiles.yaml): an override replaces which rules or
// patterns a skill renders, and install --layout <profile> renders that
// replacement, so a profile that overrides a skill down to nothing normative
// must fail here even though the base skill still passes.
func skillNormativeSections(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Skills))
	for _, skill := range idx.Skills {
		// project-router renders from router.yaml, not from composed rules or
		// patterns (skillSidecars/routerPresent already treat it specially),
		// so it legitimately carries no normative section of its own.
		if skill.ID == "project-router" {
			continue
		}
		if !skillHasNormativeSection(idx, skill) {
			problems = append(problems, fmt.Sprintf(
				"skill %q renders no normative section (must, forbidden, or validation)", skill.ID,
			))
		}
	}
	for _, profile := range idx.Profiles {
		for skillID, override := range profile.Overrides {
			skill, ok := idx.Skill(skillID)
			if !ok {
				// An unknown overridden skill is already reported by
				// profileOverrides; do not double-report it here.
				continue
			}
			overridden := applyOverride(skill, override)
			if !skillHasNormativeSection(idx, overridden) {
				problems = append(problems, fmt.Sprintf(
					"profile %q overrides skill %q to render no normative section (must, forbidden, or validation)",
					profile.ID,
					skillID,
				))
			}
		}
	}
	return problems
}

// why: an unresolved reference is already reported by
// skillCompositions/profileOverrides, so it must not double-report here.
func skillHasNormativeSection(idx *knowledge.Index, skill knowledge.Skill) bool {
	has, err := render.HasNormativeSection(idx, skill)
	return err != nil || has
}

// why: mirrors (*knowledge.Index).ApplyProfiles - a nil composition list in
// the override leaves that dimension unchanged.
func applyOverride(skill knowledge.Skill, override knowledge.SkillOverride) knowledge.Skill {
	if override.ComposeRules != nil {
		skill.ComposeRules = override.ComposeRules
	}
	if override.ComposePatterns != nil {
		skill.ComposePatterns = override.ComposePatterns
	}
	return skill
}
