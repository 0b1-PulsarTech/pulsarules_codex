package render

import "github.com/0b1-PulsarTech/pulsarules_codex/knowledge"

func installFilter(installed []string) func(string) bool {
	if len(installed) == 0 {
		return func(string) bool { return true }
	}
	set := make(map[string]bool, len(installed))
	for _, id := range installed {
		set[id] = true
	}
	return func(id string) bool { return set[id] }
}

func filterBaseline(
	baseline knowledge.RouterBaseline,
	keep func(string) bool,
) knowledge.RouterBaseline {
	return knowledge.RouterBaseline{
		Always:      filterEntries(baseline.Always, keep),
		CommitLast:  filterEntries(baseline.CommitLast, keep),
		Conditional: filterEntries(baseline.Conditional, keep),
	}
}

func filterEntries(
	entries []knowledge.RouterBaselineEntry,
	keep func(string) bool,
) []knowledge.RouterBaselineEntry {
	kept := make([]knowledge.RouterBaselineEntry, 0, len(entries))
	for _, entry := range entries {
		if keep(entry.Skill) {
			kept = append(kept, entry)
		}
	}
	return kept
}

func filterDispatch(
	rows []knowledge.RouterDispatchRow,
	keep func(string) bool,
) []knowledge.RouterDispatchRow {
	kept := make([]knowledge.RouterDispatchRow, 0, len(rows))
	for _, row := range rows {
		skills := filterIDs(row.Skills, keep)
		if len(skills) == 0 {
			continue
		}
		kept = append(kept, knowledge.RouterDispatchRow{
			Signal: row.Signal,
			Skills: skills,
			Note:   row.Note,
		})
	}
	return kept
}

func filterOrder(
	steps []knowledge.RouterOrderStep,
	keep func(string) bool,
) []knowledge.RouterOrderStep {
	kept := make([]knowledge.RouterOrderStep, 0, len(steps))
	for _, step := range steps {
		skills := filterIDs(step.Skills, keep)
		if len(skills) == 0 {
			continue
		}
		kept = append(kept, knowledge.RouterOrderStep{Skills: skills, Note: step.Note})
	}
	return kept
}

func filterIDs(ids []string, keep func(string) bool) []string {
	kept := make([]string, 0, len(ids))
	for _, id := range ids {
		if keep(id) {
			kept = append(kept, id)
		}
	}
	return kept
}
