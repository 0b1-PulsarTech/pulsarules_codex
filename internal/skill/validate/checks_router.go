package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func routerPresent(idx *knowledge.Index) []string {
	if idx.SkillExists(knowledge.RouterSkillID) {
		return nil
	}
	return []string{"missing mandatory " + knowledge.RouterSkillID + " skill"}
}

type idGroup struct {
	name string
	ids  []string
}

func routerSkillIDs(idx *knowledge.Index) []string {
	router := idx.Router
	groups := append(make([]idGroup, 0, len(router.Dispatch)+len(router.Order)),
		idGroup{"baseline always", baselineIDs(router.Baseline.Always)},
		idGroup{"baseline commit_last", baselineIDs(router.Baseline.CommitLast)},
		idGroup{"baseline conditional", baselineIDs(router.Baseline.Conditional)},
	)
	for _, row := range router.Dispatch {
		groups = append(groups, idGroup{"dispatch", row.Skills})
	}
	for _, step := range router.Order {
		groups = append(groups, idGroup{"order", step.Skills})
	}
	var problems []string
	for _, group := range groups {
		for _, id := range group.ids {
			if !idx.SkillExists(id) {
				problems = append(problems, fmt.Sprintf(
					"router %s names unknown skill %q", group.name, id,
				))
			}
		}
	}
	return problems
}

func baselineIDs(entries []knowledge.RouterBaselineEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.Skill)
	}
	return ids
}
