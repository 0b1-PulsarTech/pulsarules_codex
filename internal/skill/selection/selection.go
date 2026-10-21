package selection

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type Choice struct {
	All        bool
	RouterOnly bool
	IDs        []string
	Languages  detect.Set
}

type DependencyPull struct {
	Skill      string
	RequiredBy string
}

func (sel Choice) Resolve(idx *knowledge.Index) (ids []string, pulled []DependencyPull) {
	if sel.RouterOnly {
		return []string{knowledge.RouterSkillID}, nil
	}

	byID := make(map[string]knowledge.Skill, len(idx.Skills))
	want := make(map[string]bool, len(idx.Skills))
	for _, skill := range idx.Skills {
		byID[skill.ID] = skill
		if skill.AlwaysLoad && sel.languageEligible(idx, skill) {
			want[skill.ID] = true
		}
	}

	if sel.All {
		markAll(idx, want)
	} else {
		pulled = markSelected(byID, want, sel.IDs)
	}

	return wantedInOrder(idx, want), pulled
}

func (sel Choice) languageEligible(idx *knowledge.Index, skill knowledge.Skill) bool {
	// Empty means unknown: a repo without markers never drops its tagged baseline.
	return sel.Languages.Empty() || idx.Fits(skill.ID, sel.Languages.Strings()...)
}

func markAll(idx *knowledge.Index, want map[string]bool) {
	for _, skill := range idx.Skills {
		want[skill.ID] = true
	}
}

func markSelected(
	byID map[string]knowledge.Skill,
	want map[string]bool,
	ids []string,
) []DependencyPull {
	for _, id := range ids {
		want[id] = true
	}
	return pullDependencies(byID, want, ids)
}

func wantedInOrder(idx *knowledge.Index, want map[string]bool) []string {
	ordered := idx.SkillsOrdered()
	ids := make([]string, 0, len(want))
	for _, skill := range ordered {
		if want[skill.ID] {
			ids = append(ids, skill.ID)
		}
	}
	return ids
}

// simplification: only the selection's own deps are walked; no always-load skill but
// project-router declares compose_skills. Upgrade path: add a mandatory skill's deps to roots
// once one declares compose_skills.
func pullDependencies(
	byID map[string]knowledge.Skill,
	want map[string]bool,
	roots []string,
) []DependencyPull {
	var pulled []DependencyPull
	visited := make(map[string]bool, len(byID))
	var walk func(id string)
	walk = func(id string) {
		if visited[id] || id == knowledge.RouterSkillID {
			return
		}
		visited[id] = true
		for _, dep := range byID[id].ComposeSkills {
			if !want[dep] {
				want[dep] = true
				pulled = append(pulled, DependencyPull{Skill: dep, RequiredBy: id})
			}
			walk(dep)
		}
	}
	for _, id := range roots {
		walk(id)
	}
	return pulled
}
