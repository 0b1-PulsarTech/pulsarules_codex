package message

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

func validateDescription(
	msg commitmsg.Message, cfg ruleConfig, reporters ruleReporters,
) []finding.Finding {
	var findings []finding.Finding
	descRunes := []rune(msg.Description)

	if len(descRunes) == 0 {
		findings = append(
			findings,
			reporters.descRequired.New("commit message must have a description"),
		)
		return findings
	}

	if len(descRunes) > cfg.MaxSubjectLen {
		findings = append(findings, reporters.descLength.New(
			"subject is "+strconv.Itoa(len(descRunes))+" chars, max "+
				strconv.Itoa(cfg.MaxSubjectLen),
		))
	}

	if !unicode.IsUpper(descRunes[0]) && !msg.IsWIP {
		findings = append(
			findings,
			reporters.descCapitalize.New("description must start with a capital letter"),
		)
	}

	if descRunes[len(descRunes)-1] == '.' {
		findings = append(
			findings,
			reporters.descNoPeriod.New("description must not end with a period"),
		)
	}

	return findings
}

func validateBody(
	msg commitmsg.Message,
	cfg ruleConfig,
	reporters ruleReporters,
) []finding.Finding {
	if msg.Body == "" {
		return nil
	}
	var findings []finding.Finding
	for lineIndex, line := range strings.Split(msg.Body, "\n") {
		if len([]rune(line)) > cfg.MaxBodyLineLen {
			findings = append(findings, reporters.bodyLength.New(
				"body line "+strconv.Itoa(lineIndex+1)+" is "+strconv.Itoa(
					len([]rune(line)),
				)+" chars, max "+strconv.Itoa(
					cfg.MaxBodyLineLen,
				),
			))
		}
	}
	if total := len([]rune(msg.Body)); total > cfg.MaxBodyLen {
		findings = append(findings, reporters.bodyTotalLength.New(
			"body is "+strconv.Itoa(total)+" chars, max "+strconv.Itoa(
				cfg.MaxBodyLen,
			)+"; keep it short or use a subject-only commit",
		))
	}
	return findings
}

func validateToolTrailers(msg commitmsg.Message, reporters ruleReporters) []finding.Finding {
	trailer := msg.ToolTrailer()
	if trailer == "" {
		return nil
	}
	return []finding.Finding{
		reporters.noCoauthor.New(
			"tool-attribution trailer '" + trailer + "' is not allowed; " +
				"remove it before committing",
		),
	}
}
