package commit

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func validateDescription(msg commitmsg.Message, cfg RuleConfig) []core.Finding {
	var findings []core.Finding
	desc := msg.Description

	if desc == "" {
		findings = append(
			findings,
			descRequiredReporter.New("commit message must have a description"),
		)
		return findings
	}

	if len([]rune(desc)) > cfg.MaxSubjectLen {
		findings = append(findings, descLengthReporter.New(
			"subject is "+strconv.Itoa(
				len([]rune(desc)),
			)+" chars, max "+strconv.Itoa(
				cfg.MaxSubjectLen,
			),
		))
	}

	first := []rune(desc)[0]
	if !unicode.IsUpper(first) && !msg.IsWIP {
		findings = append(
			findings,
			descCapitalizeReporter.New("description must start with a capital letter"),
		)
	}

	last := []rune(desc)[len([]rune(desc))-1]
	if last == '.' {
		findings = append(
			findings,
			descNoPeriodReporter.New("description must not end with a period"),
		)
	}

	return findings
}

func validateBody(msg commitmsg.Message, cfg RuleConfig) []core.Finding {
	if msg.Body == "" {
		return nil
	}
	var findings []core.Finding
	for i, line := range strings.Split(msg.Body, "\n") {
		if len([]rune(line)) > cfg.MaxBodyLineLen {
			findings = append(findings, bodyLengthReporter.New(
				"body line "+strconv.Itoa(i+1)+" is "+strconv.Itoa(
					len([]rune(line)),
				)+" chars, max "+strconv.Itoa(
					cfg.MaxBodyLineLen,
				),
			))
		}
	}
	if total := len([]rune(msg.Body)); total > cfg.MaxBodyLen {
		findings = append(findings, bodyTotalLengthReporter.New(
			"body is "+strconv.Itoa(total)+" chars, max "+strconv.Itoa(
				cfg.MaxBodyLen,
			)+"; keep it short or use a subject-only commit",
		))
	}
	return findings
}

func validateToolTrailers(msg commitmsg.Message, cfg RuleConfig) []core.Finding {
	if !cfg.RejectToolTrailers {
		return nil
	}
	trailer := msg.ToolTrailer()
	if trailer == "" {
		return nil
	}
	return []core.Finding{
		noCoauthorReporter.New(
			"tool-attribution trailer '" + trailer + "' is not allowed; remove it before committing",
		),
	}
}

func isScopeChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' ||
		r == '_'
}
