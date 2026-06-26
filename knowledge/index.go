package knowledge

import (
	"cmp"
	"slices"
)

type Index struct {
	Rules     []Rule
	Patterns  []Pattern
	Workflows []Workflow
	Skills    []Skill
}

func (idx *Index) SkillsOrdered() []Skill {
	ordered := make([]Skill, len(idx.Skills))
	copy(ordered, idx.Skills)
	slices.SortFunc(ordered, func(left, right Skill) int {
		return cmp.Or(cmp.Compare(left.Order, right.Order), cmp.Compare(left.ID, right.ID))
	})
	return ordered
}
