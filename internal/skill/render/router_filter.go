package render

import "github.com/0b1-PulsarTech/pulsarules_codex/knowledge"

// installFilter returns a predicate reporting whether a skill id should appear
// in the rendered router. An empty installed list means no filtering (every
// skill is kept), so a full or router-only install renders the complete router;
// a non-empty list keeps only the ids it contains.
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

// filterBaseline keeps only the baseline entries whose skill is installed.
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

// filterDispatch trims each row to its installed skills and drops a row whose
// skills are all uninstalled.
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

// filterOrder trims each step to its installed skills and drops an empty step.
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
