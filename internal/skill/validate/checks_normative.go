package validate

import (
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// skillNormativeSections reports every skill whose composed rules/patterns
// render no non-empty "must", "forbidden", or "validation" section:
// documentation, not a governed contract, that would stay green while an
// obligation disappears. It checks base composition and every profile's
// override (profiles.yaml), since --layout can drop normative content.
func skillNormativeSections(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Skills))
	for _, skill := range idx.Skills {
		// project-router renders from router.yaml, not from composed rules or
		// patterns (skillSidecars/routerPresent already treat it specially),
		// so it legitimately carries no normative section of its own.
		if skill.ID == "project-router" {
			continue
		}
		has, problem := skillHasNormativeSection(idx, skill)
		switch {
		case problem != "":
			problems = append(problems, problem)
		case !has:
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
			has, problem := skillHasNormativeSection(idx, overridden)
			switch {
			case problem != "":
				problems = append(problems, fmt.Sprintf("profile %q: %s", profile.ID, problem))
			case !has:
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

// why: an unresolved reference is already reported by skillCompositions/profileOverrides, so it
// must not double-report here - but that is the ONLY failure worth swallowing. This used to
// swallow every error, which silently passed a skill whose {{define "must"}} block does not parse:
// a real defect nothing else in this function's own pipeline reports.
func skillHasNormativeSection(
	idx *knowledge.Index,
	skill knowledge.Skill,
) (has bool, problem string) {
	var err error
	has, err = render.HasNormativeSection(idx, skill)
	switch {
	case errors.Is(err, render.ErrUnknownComposition):
		return true, ""
	case err != nil:
		return true, err.Error()
	default:
		return has, ""
	}
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
