package install

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/selection"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func resolveSelection(opts *version.Options, idx *knowledge.Index) (selection.Choice, error) {
	switch {
	case opts.RouterOnly:
		return selection.Choice{RouterOnly: true}, nil
	case opts.All:
		return selection.Choice{All: true}, nil
	case opts.Skills != "":
		return selection.Choice{IDs: splitCSV(opts.Skills)}, nil
	case stdinIsTerminal():
		return promptSkillSelection(idx)
	// A non-TTY fails: prompting would hang CI, and installing everything would
	// ignore the caller's choice.
	default:
		return selection.Choice{}, fmt.Errorf(
			"no --all/--skills/--router-only given and stdin is not a terminal: " +
				"pass --all to install every skill, or --skills a,b,c to choose",
		)
	}
}

func printSkillCatalog(out io.Writer, catalog []knowledge.Skill) {
	_, _ = fmt.Fprintln(out, "Select skills to install (mandatory skills are always included):")
	for pos, skill := range catalog {
		mark, note := " ", ""
		if skill.AlwaysLoad {
			// Mandatory skills show pre-selected and locked, so the user sees the whole
			// baseline.
			mark, note = "x", " (mandatory, locked)"
		}
		_, _ = fmt.Fprintf(
			out,
			"  %2d. [%s] %s - %s%s\n",
			pos+1,
			mark,
			printable.Line(skill.ID),
			printable.Line(knowledge.FirstSentence(skill.Description)),
			note,
		)
	}
}

func promptSkillSelection(idx *knowledge.Index) (selection.Choice, error) {
	catalog := idx.SkillsOrdered()
	printSkillCatalog(os.Stdout, catalog)
	_, _ = fmt.Print("skills> (e.g. 1,4,7-9 | a = all | Enter = mandatory-only): ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')

	picked, err := parseSkillSelection(line, len(catalog))
	if err != nil {
		return selection.Choice{}, fmt.Errorf("parse skill selection: %w", err)
	}
	ids := make([]string, 0, len(picked))
	for _, i := range picked {
		ids = append(ids, catalog[i-1].ID)
	}
	// Selection.Resolve unions the mandatory skills in regardless of what is picked.
	return selection.Choice{IDs: ids}, nil
}

func parseSkillSelection(input string, total int) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		// Blank input selects nothing; the caller reads that as the mandatory-only baseline.
		return nil, nil
	}
	if strings.EqualFold(input, "a") {
		return allChoices(total), nil
	}

	seen := make(map[int]bool)
	var indices []int
	for token := range strings.SplitSeq(input, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		start, end, parseErr := parseToken(token, total)
		if parseErr != nil {
			return nil, parseErr
		}
		indices = addRange(seen, indices, start, end)
	}
	slices.Sort(indices)
	return indices, nil
}

func allChoices(total int) []int {
	all := make([]int, total)
	for slot := range all {
		all[slot] = slot + 1
	}
	return all
}

func parseToken(token string, total int) (start, end int, err error) {
	start, end, err = parseSelectionRange(token)
	if err != nil {
		return 0, 0, err
	}
	if start < 1 || end > total || start > end {
		return 0, 0, fmt.Errorf("selection %q out of range 1-%d", token, total)
	}
	return start, end, nil
}

func addRange(seen map[int]bool, indices []int, start, end int) []int {
	for choice := start; choice <= end; choice++ {
		if !seen[choice] {
			seen[choice] = true
			indices = append(indices, choice)
		}
	}
	return indices
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
