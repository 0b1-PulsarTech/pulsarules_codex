package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// profileOverrides reports any profile whose override targets an unknown skill or
// references a rule/pattern that does not exist or has an empty body. This proves
// the variant rules a profile selects are actually composable and will render, not
// silently dropped.
func profileOverrides(idx *knowledge.Index) []string {
	var problems []string
	for _, profile := range idx.Profiles {
		for skillID, override := range profile.Overrides {
			if !idx.SkillExists(skillID) {
				problems = append(problems, fmt.Sprintf(
					"profile %q overrides unknown skill %q", profile.ID, skillID,
				))
				continue
			}
			comps := []composition{
				{
					refs:    override.ComposeRules,
					resolve: func(id string) bool { return idx.RuleExists(id) && idx.Body("rules", id) != "" },
					missing: profileMissing(profile.ID, skillID, "rule"),
				},
				{
					refs:    override.ComposePatterns,
					resolve: func(id string) bool { return idx.PatternExists(id) && idx.Body("patterns", id) != "" },
					missing: profileMissing(profile.ID, skillID, "pattern"),
				},
			}
			problems = append(problems, runCompositions(comps)...)
		}
	}
	return problems
}

func profileMissing(profileID, skillID, kind string) func(string) string {
	return func(ref string) string {
		return fmt.Sprintf(
			"profile %q override of skill %q composes missing/empty %s %q",
			profileID, skillID, kind, ref,
		)
	}
}

func routerPresent(idx *knowledge.Index) []string {
	if idx.SkillExists("project-router") {
		return nil
	}
	return []string{"missing mandatory project-router skill"}
}

// skillSidecars reports every non-router skill that has no curated sidecar body.
// The sidecar carries the skill's curated Mandatory workflow, consolidated
// checklists, forbidden actions, and expected outputs; project-router renders
// from its own template and is exempt.
func skillSidecars(idx *knowledge.Index) []string {
	var problems []string
	for _, skill := range idx.Skills {
		if skill.ID == "project-router" {
			continue
		}
		if idx.Body("skills", skill.ID) == "" {
			problems = append(
				problems,
				fmt.Sprintf(
					"skill %q has no curated sidecar under standards/skills/%s.md",
					skill.ID,
					skill.ID,
				),
			)
		}
	}
	return problems
}

func missingRef(ownerID, kind string) func(string) string {
	return func(ref string) string {
		return fmt.Sprintf("skill %q composes unknown %s %q", ownerID, kind, ref)
	}
}

func emptyBody(ownerID, kind string) func(string) string {
	return func(ref string) string {
		return fmt.Sprintf("skill %q composes %s %q with empty body", ownerID, kind, ref)
	}
}

func hasBody(idx *knowledge.Index, kind string) func(string) bool {
	return func(id string) bool { return idx.Body(kind, id) != "" }
}
