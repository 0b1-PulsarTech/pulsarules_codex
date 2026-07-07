package list

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func RunList(inj remy.Injector, opts *version.Options) error {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	return writeList(os.Stdout, idx, opts.Kind)
}

func writeList(out io.Writer, idx *knowledge.Index, kind string) error {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	switch kind {
	case "skills":
		writeSkills(writer, idx)
	case "rules":
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tDEPS")
		for _, rule := range idx.Rules {
			writeRow(writer, rule.ID, rule.Name, joinOrDash(rule.Dependencies))
		}
	case "patterns":
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCOMPOSES")
		for _, pattern := range idx.Patterns {
			writeRow(writer, pattern.ID, pattern.Name, joinOrDash(pattern.Composes))
		}
	case "workflows":
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCOMPOSES")
		for _, workflow := range idx.Workflows {
			composes := slices.Concat(workflow.ComposesRules, workflow.ComposesPatterns)
			writeRow(writer, workflow.ID, workflow.Name, joinOrDash(composes))
		}
	default:
		return fmt.Errorf("unknown list kind %q (skills|rules|patterns|workflows)", kind)
	}
	if flushErr := writer.Flush(); flushErr != nil {
		return fmt.Errorf("flush table: %w", flushErr)
	}
	return nil
}

func writeSkills(writer io.Writer, idx *knowledge.Index) {
	_, _ = fmt.Fprintln(writer, "SKILL\tLOAD\tORDER\tDESCRIPTION")
	for _, skill := range idx.SkillsOrdered() {
		load := "signal"
		if skill.AlwaysLoad {
			load = "always"
		}
		writeRow(
			writer,
			skill.ID,
			load,
			strconv.Itoa(skill.Order),
			knowledge.FirstSentence(skill.Description),
		)
	}
}

func writeRow(writer io.Writer, cells ...string) {
	_, _ = fmt.Fprintln(writer, strings.Join(cells, "\t"))
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ", ")
}
