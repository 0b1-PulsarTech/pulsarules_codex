package controlflow

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

const (
	forbidElseMessage = "else branch; return early, start from a default and override it " +
		"in one if, or use a switch expression"
	forbidElseSuggestion = "invert the condition and return early, or assign a default first"
)

func forbidSyntaxElses(
	fc pass.FileChange,
	file *syntax.File,
	reporter finding.Reporter,
) []finding.Finding {
	findings := make([]finding.Finding, 0, len(file.Elses))
	for _, branch := range file.Elses {
		findings = append(findings, reporter.At(
			fc.Path, branch.Line, forbidElseMessage, forbidElseSuggestion,
		))
	}
	return findings
}

func exitSyntaxElses(
	fc pass.FileChange,
	file *syntax.File,
	reporter finding.Reporter,
) []finding.Finding {
	var findings []finding.Finding
	for _, branch := range file.Elses {
		if !branch.AfterExit {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path, branch.Line,
			"else after a branch that already returns, throws, breaks or continues; drop the else",
			"remove the else and dedent its body",
		))
	}
	return findings
}
