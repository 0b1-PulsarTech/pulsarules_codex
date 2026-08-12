package validate

import (
	"fmt"
	"strings"

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

// minOrientationChars is how much prose a sidecar must carry once its pointer
// to the composed rule is discounted. The hand-written sidecars clear it many
// times over; a body slimmed to the pointer alone leaves nothing.
const minOrientationChars = 120

// skillSidecars reports every non-router skill whose sidecar carries no
// orientation. The sidecar says what the skill governs and when to reach for
// it - the normative clauses live in the composed rules and patterns - and it
// is the first thing an agent reads to decide whether the skill applies.
// project-router renders from its own template and is exempt.
func skillSidecars(idx *knowledge.Index) []string {
	var problems []string
	for _, skill := range idx.Skills {
		if skill.ID == "project-router" {
			continue
		}
		body := idx.Body("skills", skill.ID)
		if body == "" {
			problems = append(problems, fmt.Sprintf(
				"skill %q has no curated sidecar under standards/skills/%s.md", skill.ID, skill.ID,
			))
			continue
		}
		if len(orientationProse(body)) < minOrientationChars {
			problems = append(problems, fmt.Sprintf(
				"skill %q sidecar states no orientation, only a pointer to its composed rule",
				skill.ID,
			))
		}
	}
	return problems
}

// why: the pointer names the composed rule and says nothing about when to
// reach for the skill, so it must not count towards the orientation.
func orientationProse(body string) string {
	var prose strings.Builder
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "the composed") {
			continue
		}
		prose.WriteString(trimmed)
	}
	return prose.String()
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
