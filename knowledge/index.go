package knowledge

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/catalog"
)

type Index struct {
	Rules      []Rule
	Patterns   []Pattern
	Workflows  []Workflow
	Skills     []Skill
	Profiles   []Profile
	References []Reference
	Router     RouterSpec

	rules      catalog.Catalog[Rule]
	patterns   catalog.Catalog[Pattern]
	workflows  catalog.Catalog[Workflow]
	skills     catalog.Catalog[Skill]
	profiles   catalog.Catalog[Profile]
	references catalog.Catalog[Reference]
	bodies     map[string]string
	summaries  map[string]string
}

func (idx *Index) Rule(id string) (Rule, bool) { return idx.rules.Get(id) }

func (idx *Index) Pattern(id string) (Pattern, bool) { return idx.patterns.Get(id) }

func (idx *Index) Workflow(id string) (Workflow, bool) { return idx.workflows.Get(id) }

func (idx *Index) Skill(id string) (Skill, bool) { return idx.skills.Get(id) }

func (idx *Index) Reference(id string) (Reference, bool) { return idx.references.Get(id) }

func (idx *Index) ReferenceExists(id string) bool { return idx.references.Has(id) }

func (idx *Index) RuleExists(id string) bool { return idx.rules.Has(id) }

func (idx *Index) PatternExists(id string) bool { return idx.patterns.Has(id) }

func (idx *Index) WorkflowExists(id string) bool { return idx.workflows.Has(id) }

func (idx *Index) SkillExists(id string) bool { return idx.skills.Has(id) }

func (idx *Index) ApplyProfiles(ids []string) error {
	for _, id := range ids {
		profile, ok := idx.profiles.Get(id)
		if !ok {
			return fmt.Errorf("unknown profile %q", id)
		}
		for skillID, override := range profile.Overrides {
			skill, ok := idx.skills.Get(skillID)
			if !ok {
				return fmt.Errorf("profile %q overrides unknown skill %q", id, skillID)
			}
			override.Apply(&skill)
			idx.skills.Put(skillID, skill)
			for pos := range idx.Skills {
				if idx.Skills[pos].ID == skillID {
					idx.Skills[pos] = skill
				}
			}
		}
	}
	return nil
}

func (idx *Index) Body(kind Kind, id string) string { return idx.bodies[kind.key(id)] }

func (idx *Index) Summary(kind Kind, id string) string { return idx.summaries[kind.key(id)] }

func (idx *Index) SkillsOrdered() []Skill {
	ordered := make([]Skill, len(idx.Skills))
	copy(ordered, idx.Skills)
	slices.SortFunc(ordered, func(left, right Skill) int {
		return cmp.Or(cmp.Compare(left.Order, right.Order), cmp.Compare(left.ID, right.ID))
	})
	return ordered
}

func (idx *Index) BaselineFor(languages ...string) []string {
	if idx == nil {
		return nil
	}
	var ids []string
	for _, entry := range idx.Router.Baseline.Always {
		if idx.Fits(entry.Skill, languages...) {
			ids = append(ids, entry.Skill)
		}
	}
	return ids
}

func (idx *Index) Fits(id string, languages ...string) bool {
	for _, skill := range idx.Skills {
		if skill.ID != id {
			continue
		}
		// A skill with no languages is neutral and fits every project.
		return len(skill.Languages) == 0 ||
			slices.ContainsFunc(skill.Languages, func(own string) bool {
				return slices.Contains(languages, own)
			})
	}
	return false
}
