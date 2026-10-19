package validate

import (
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/render"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func skillNormativeSections(idx *knowledge.Index) []string {
	problems := make([]string, 0, len(idx.Skills))
	for _, skill := range idx.Skills {
		// The project-router skill renders from router.yaml, not from composed rules or
		// patterns (skillSidecars/routerPresent already treat it specially),
		// so it legitimately carries no normative section of its own.
		if skill.ID == knowledge.RouterSkillID {
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
	// A --layout profile must not override a skill into stating no obligation.
	for _, profile := range idx.Profiles {
		for skillID, override := range profile.Overrides {
			skill, ok := idx.Skill(skillID)
			if !ok {
				// An unknown overridden skill is already reported by
				// profileOverrides; do not double-report it here.
				continue
			}
			overridden := skill
			override.Apply(&overridden)
			has, problem := skillHasNormativeSection(idx, overridden)
			switch {
			case problem != "":
				problems = append(problems, fmt.Sprintf("profile %q: %s", profile.ID, problem))
			case !has:
				problems = append(problems, fmt.Sprintf(
					"profile %q overrides skill %q to render no normative section "+
						"(must, forbidden, or validation)",
					profile.ID,
					skillID,
				))
			}
		}
	}
	return problems
}

func skillHasNormativeSection(
	idx *knowledge.Index,
	skill knowledge.Skill,
) (has bool, problem string) {
	var err error
	has, err = render.HasNormativeSection(idx, skill)
	switch {
	// An unresolved reference is reported elsewhere; any other error is surfaced here.
	case errors.Is(err, render.ErrUnknownComposition):
		return true, ""
	case err != nil:
		return true, err.Error()
	default:
		return has, ""
	}
}
