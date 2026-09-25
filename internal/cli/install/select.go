package install

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/output"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func resolveSelection(opts *cliopts.Options, idx *knowledge.Index) (output.Selection, error) {
	switch {
	case opts.RouterOnly:
		return output.Selection{RouterOnly: true}, nil
	case opts.All:
		return output.Selection{All: true}, nil
	case opts.Skills != "":
		return output.Selection{IDs: splitCSV(opts.Skills)}, nil
	case stdinIsTerminal():
		return promptSkillSelection(idx)
	// A non-TTY fails: prompting would hang CI, and installing everything would
	// ignore the caller's choice.
	default:
		return output.Selection{}, fmt.Errorf(
			"no --all/--skills/--router-only given and stdin is not a terminal: " +
				"pass --all to install every skill, or --skills a,b,c to choose",
		)
	}
}

func promptSkillSelection(idx *knowledge.Index) (output.Selection, error) {
	catalog := idx.SkillsOrdered()
	_, _ = fmt.Println("Select skills to install (mandatory skills are always included):")
	for i, skill := range catalog {
		mark, note := " ", ""
		if skill.AlwaysLoad {
			// Mandatory skills show pre-selected and locked, so the user sees the whole
			// baseline.
			mark, note = "x", " (mandatory, locked)"
		}
		_, _ = fmt.Printf(
			"  %2d. [%s] %s - %s%s\n",
			i+1, mark, skill.ID, knowledge.FirstSentence(skill.Description), note,
		)
	}
	_, _ = fmt.Print("skills> (e.g. 1,4,7-9 | a = all | Enter = mandatory-only): ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')

	picked, err := parseSkillSelection(line, len(catalog))
	if err != nil {
		return output.Selection{}, fmt.Errorf("parse skill selection: %w", err)
	}
	ids := make([]string, 0, len(picked))
	for _, i := range picked {
		ids = append(ids, catalog[i-1].ID)
	}
	// Selection.Resolve unions the mandatory skills in regardless of what is picked.
	return output.Selection{IDs: ids}, nil
}

func parseSkillSelection(input string, total int) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		// Blank input selects nothing; the caller reads that as the mandatory-only baseline.
		return nil, nil
	}
	if strings.EqualFold(input, "a") {
		all := make([]int, total)
		for slot := range all {
			all[slot] = slot + 1
		}
		return all, nil
	}

	seen := make(map[int]bool)
	var indices []int
	for token := range strings.SplitSeq(input, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		start, end, parseErr := parseSelectionRange(token)
		if parseErr != nil {
			return nil, parseErr
		}
		if start < 1 || end > total || start > end {
			return nil, fmt.Errorf("selection %q out of range 1-%d", token, total)
		}
		for choice := start; choice <= end; choice++ {
			if !seen[choice] {
				seen[choice] = true
				indices = append(indices, choice)
			}
		}
	}
	slices.Sort(indices)
	return indices, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func parseSelectionRange(token string) (start, end int, err error) {
	before, after, isRange := strings.Cut(token, "-")
	if !isRange {
		num, convErr := strconv.Atoi(token)
		if convErr != nil {
			return 0, 0, fmt.Errorf("parse selection %q: %w", token, convErr)
		}
		return num, num, nil
	}
	start, err = strconv.Atoi(strings.TrimSpace(before))
	if err != nil {
		return 0, 0, fmt.Errorf("parse range start %q: %w", token, err)
	}
	end, err = strconv.Atoi(strings.TrimSpace(after))
	if err != nil {
		return 0, 0, fmt.Errorf("parse range end %q: %w", token, err)
	}
	return start, end, nil
}
