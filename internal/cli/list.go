package cli

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func runList(inj remy.Injector, opts *cliopts.Options) error {
	idx, err := remy.Get[*knowledge.Index](inj)
	if err != nil {
		return fmt.Errorf("get knowledge index: %w", err)
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	switch opts.Kind {
	case "skills":
		_, _ = fmt.Fprintln(writer, "SKILL\tLOAD\tORDER\tDESCRIPTION")
		for _, skill := range idx.SkillsOrdered() {
			load := "signal"
			if skill.AlwaysLoad {
				load = "always"
			}
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%d\t%s\n",
				skill.ID,
				load,
				skill.Order,
				knowledge.FirstSentence(skill.Description),
			)
		}
	case "rules":
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tDEPS")
		for _, rule := range idx.Rules {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\n",
				rule.ID,
				rule.Name,
				joinOrDash(rule.Dependencies),
			)
		}
	case "patterns":
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCOMPOSES")
		for _, pattern := range idx.Patterns {
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\n",
				pattern.ID,
				pattern.Name,
				joinOrDash(pattern.Composes),
			)
		}
	case "workflows":
		_, _ = fmt.Fprintln(writer, "ID\tNAME\tCOMPOSES")
		for _, workflow := range idx.Workflows {
			composes := slices.Concat(workflow.ComposesRules, workflow.ComposesPatterns)
			_, _ = fmt.Fprintf(
				writer,
				"%s\t%s\t%s\n",
				workflow.ID,
				workflow.Name,
				joinOrDash(composes),
			)
		}
	default:
		return fmt.Errorf("unknown list kind %q (skills|rules|patterns|workflows)", opts.Kind)
	}
	if flushErr := writer.Flush(); flushErr != nil {
		return fmt.Errorf("flush table: %w", flushErr)
	}
	return nil
}

func joinOrDash(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	var builder strings.Builder
	for i, item := range items {
		if i > 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(item)
	}
	return builder.String()
}
