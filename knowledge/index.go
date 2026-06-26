package knowledge

import (
	"fmt"
	"slices"
)

// Index is the parsed knowledge base: typed items, by-id maps for cross-reference
// resolution, and the markdown bodies of rules/patterns/workflows for transclusion.
type Index struct {
	Rules      []Rule
	Patterns   []Pattern
	Workflows  []Workflow
	Skills     []Skill
	Profiles   []Profile
	References []Reference
	Router     RouterSpec

	ruleByID      map[string]Rule
	patternByID   map[string]Pattern
	workflowByID  map[string]Workflow
	skillByID     map[string]Skill
	profileByID   map[string]Profile
	referenceByID map[string]Reference
	bodies        map[string]string // "<kind>:<id>" -> markdown body
}

// Rule returns the rule with the given id.
func (i *Index) Rule(id string) (Rule, bool) { r, ok := i.ruleByID[id]; return r, ok }

// Pattern returns the pattern with the given id.
func (i *Index) Pattern(id string) (Pattern, bool) { p, ok := i.patternByID[id]; return p, ok }

// Workflow returns the workflow with the given id.
func (i *Index) Workflow(id string) (Workflow, bool) { w, ok := i.workflowByID[id]; return w, ok }

// Skill returns the skill with the given id.
func (i *Index) Skill(id string) (Skill, bool) { s, ok := i.skillByID[id]; return s, ok }

// Reference returns the external reference with the given id.
func (i *Index) Reference(id string) (Reference, bool) {
	r, ok := i.referenceByID[id]
	return r, ok
}

// ReferenceExists reports whether a reference with the given id is loaded.
func (i *Index) ReferenceExists(id string) bool { _, ok := i.referenceByID[id]; return ok }

// ApplyProfiles rewrites the composition lists of the skills targeted by the named
// profiles, in place, so a subsequent render composes the variant rules/patterns.
// It returns an error for an unknown profile or an override of an unknown skill;
// composed-id resolution is still verified at render time.
func (i *Index) ApplyProfiles(ids []string) error {
	for _, id := range ids {
		profile, ok := i.profileByID[id]
		if !ok {
			return fmt.Errorf("unknown profile %q", id)
		}
		for skillID, override := range profile.Overrides {
			skill, ok := i.skillByID[skillID]
			if !ok {
				return fmt.Errorf("profile %q overrides unknown skill %q", id, skillID)
			}
			if override.ComposeRules != nil {
				skill.ComposeRules = override.ComposeRules
			}
			if override.ComposePatterns != nil {
				skill.ComposePatterns = override.ComposePatterns
			}
			i.skillByID[skillID] = skill
			for idx := range i.Skills {
				if i.Skills[idx].ID == skillID {
					i.Skills[idx] = skill
				}
			}
		}
	}
	return nil
}

// RuleExists reports whether a rule with the given id is loaded.
func (i *Index) RuleExists(id string) bool { _, ok := i.ruleByID[id]; return ok }

// PatternExists reports whether a pattern with the given id is loaded.
func (i *Index) PatternExists(id string) bool { _, ok := i.patternByID[id]; return ok }

// WorkflowExists reports whether a workflow with the given id is loaded.
func (i *Index) WorkflowExists(id string) bool { _, ok := i.workflowByID[id]; return ok }

// SkillExists reports whether a skill with the given id is loaded.
func (i *Index) SkillExists(id string) bool { _, ok := i.skillByID[id]; return ok }

// Body returns the markdown body of a rule/pattern/workflow, without frontmatter.
func (i *Index) Body(kind, id string) string { return i.bodies[kind+":"+id] }

// SkillsOrdered returns skills sorted by Order then ID, so the router and
// listings present them in composition order.
func (i *Index) SkillsOrdered() []Skill {
	ordered := make([]Skill, len(i.Skills))
	copy(ordered, i.Skills)
	slices.SortFunc(ordered, func(a, b Skill) int {
		if a.Order != b.Order {
			if a.Order < b.Order {
				return -1
			}
			return 1
		}
		if a.ID < b.ID {
			return -1
		} else if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return ordered
}
