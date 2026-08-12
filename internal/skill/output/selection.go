package output

import "github.com/0b1-PulsarTech/pulsarules_codex/knowledge"

// Selection resolves a user request into the skill ids to install.
type Selection struct {
	All        bool
	RouterOnly bool
	IDs        []string
}

// DependencyPull names a skill Resolve added beyond the caller's selection
// because another selected skill's compose_skills listed it, and which skill
// asked for it, so the caller can report why the install grew.
type DependencyPull struct {
	Skill      string
	RequiredBy string
}

// Resolve returns the concrete skill id list for the selection against idx,
// in composition order, plus every skill pulled in transitively through a
// compose_skills reference. Every mode except RouterOnly carries the
// mandatory baseline (project-router plus every always_load skill), so an
// empty selection resolves to that baseline alone; RouterOnly skips it.
func (sel Selection) Resolve(idx *knowledge.Index) (ids []string, pulled []DependencyPull) {
	if sel.RouterOnly {
		return []string{"project-router"}, nil
	}

	byID := make(map[string]knowledge.Skill, len(idx.Skills))
	want := make(map[string]bool, len(idx.Skills))
	for _, skill := range idx.Skills {
		byID[skill.ID] = skill
		if skill.AlwaysLoad {
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

// why: compose_skills on project-router lists the full router-doc catalog,
// not a dependency, so it's exempt; a visited set stops a hand-authored cycle.
// simplification: only the selection's own deps are walked, since only
// project-router uses compose_skills today. Upgrade path: revisit if a
// mandatory skill adds one, and add it to roots.
func pullDependencies(
	byID map[string]knowledge.Skill,
	want map[string]bool,
	roots []string,
) []DependencyPull {
	var pulled []DependencyPull
	visited := make(map[string]bool, len(byID))
	var walk func(id string)
	walk = func(id string) {
		if visited[id] || id == "project-router" {
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
