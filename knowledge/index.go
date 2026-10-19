package knowledge

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/catalog"
)

type Index struct {
	Rules     []Rule
	Patterns  []Pattern
	Workflows []Workflow
	Skills    []Skill
	Profiles  []Profile
	Router    RouterSpec

	rules      catalog.Catalog[Rule]
	patterns   catalog.Catalog[Pattern]
	workflows  catalog.Catalog[Workflow]
	skills     catalog.Catalog[Skill]
	profiles   catalog.Catalog[Profile]
	references catalog.Catalog[Reference]
	bodies     map[string]string
	summaries  map[string]string
}

func NewIndex(skills []Skill) (*Index, error) {
	idx := newIndex()
	for _, skill := range skills {
		if !idx.skills.Add(skill.ID, skill) {
			return nil, fmt.Errorf("duplicate skill id %q", skill.ID)
		}
	}
	idx.publish()
	return idx, nil
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
	defer idx.publish()
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
			idx.skills.Replace(skillID, skill)
		}
	}
	return nil
}

func (idx *Index) publish() {
	idx.Rules = slices.Collect(idx.rules.All())
	idx.Patterns = slices.Collect(idx.patterns.All())
	idx.Workflows = slices.Collect(idx.workflows.All())
	idx.Skills = slices.Collect(idx.skills.All())
	idx.Profiles = slices.Collect(idx.profiles.All())
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
	skill, ok := idx.skills.Get(id)
	if !ok {
		return false
	}
	// A skill with no languages is neutral and fits every project.
	return len(skill.Languages) == 0 ||
		slices.ContainsFunc(skill.Languages, func(own string) bool {
			return slices.Contains(languages, own)
		})
}
