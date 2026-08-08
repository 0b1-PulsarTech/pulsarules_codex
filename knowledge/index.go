package knowledge

import (
	"cmp"
	"fmt"
	"slices"
)

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
	bodies        map[string]string
	summaries     map[string]string
}

func (idx *Index) Rule(id string) (Rule, bool) { rule, ok := idx.ruleByID[id]; return rule, ok }

func (idx *Index) Pattern(id string) (Pattern, bool) {
	pattern, ok := idx.patternByID[id]
	return pattern, ok
}

func (idx *Index) Workflow(id string) (Workflow, bool) {
	workflow, ok := idx.workflowByID[id]
	return workflow, ok
}

func (idx *Index) Skill(id string) (Skill, bool) {
	skill, ok := idx.skillByID[id]
	return skill, ok
}

func (idx *Index) Reference(id string) (Reference, bool) {
	ref, ok := idx.referenceByID[id]
	return ref, ok
}

func (idx *Index) ReferenceExists(id string) bool { _, ok := idx.referenceByID[id]; return ok }

func (idx *Index) ApplyProfiles(ids []string) error {
	for _, id := range ids {
		profile, ok := idx.profileByID[id]
		if !ok {
			return fmt.Errorf("unknown profile %q", id)
		}
		for skillID, override := range profile.Overrides {
			skill, ok := idx.skillByID[skillID]
			if !ok {
				return fmt.Errorf("profile %q overrides unknown skill %q", id, skillID)
			}
			if override.ComposeRules != nil {
				skill.ComposeRules = override.ComposeRules
			}
			if override.ComposePatterns != nil {
				skill.ComposePatterns = override.ComposePatterns
			}
			idx.skillByID[skillID] = skill
			for pos := range idx.Skills {
				if idx.Skills[pos].ID == skillID {
					idx.Skills[pos] = skill
				}
			}
		}
	}
	return nil
}

func (idx *Index) RuleExists(id string) bool { _, ok := idx.ruleByID[id]; return ok }

func (idx *Index) PatternExists(id string) bool { _, ok := idx.patternByID[id]; return ok }

func (idx *Index) WorkflowExists(id string) bool { _, ok := idx.workflowByID[id]; return ok }

func (idx *Index) SkillExists(id string) bool { _, ok := idx.skillByID[id]; return ok }

func (idx *Index) Body(kind, id string) string { return idx.bodies[kind+":"+id] }

func (idx *Index) Summary(kind, id string) string { return idx.summaries[kind+":"+id] }

func (idx *Index) SkillsOrdered() []Skill {
	ordered := make([]Skill, len(idx.Skills))
	copy(ordered, idx.Skills)
	slices.SortFunc(ordered, func(left, right Skill) int {
		return cmp.Or(cmp.Compare(left.Order, right.Order), cmp.Compare(left.ID, right.ID))
	})
	return ordered
}
