package golangcilint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type lintReport struct {
	Issues []lintIssue `json:"Issues"`
}

type lintIssue struct {
	FromLinter string  `json:"FromLinter"`
	Text       string  `json:"Text"`
	Severity   string  `json:"Severity"`
	Pos        lintPos `json:"Pos"`
}

type lintPos struct {
	Filename string `json:"Filename"`
	Line     int    `json:"Line"`
}

func (r *Runner) parseOutput(out []byte, cmdErr error) []finding.Finding {
	if cmdErr != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](cmdErr); ok {
			if !isLintExit(exitErr.ExitCode()) {
				return []finding.Finding{
					r.failure.New(
						fmt.Sprintf("golangci-lint failed: %s", string(exitErr.Stderr)),
					),
				}
			}
		}
	}

	jsonOut := extractJSON(out)

	var result lintReport

	if parseErr := json.Unmarshal(jsonOut, &result); parseErr != nil {
		return []finding.Finding{
			r.failure.New(
				fmt.Sprintf("failed to parse golangci-lint output: %s", parseErr),
			),
		}
	}

	var findings []finding.Finding
	for _, issue := range result.Issues {
		sev := finding.SeverityWarning
		if strings.Contains(issue.Severity, "error") {
			sev = finding.SeverityError
		}
		findings = append(findings, finding.Finding{
			AnalyzerID: fmt.Sprintf("golangci-lint/%s", issue.FromLinter),
			Severity:   sev,
			Category:   finding.CatSyntax,
			File:       issue.Pos.Filename,
			Line:       issue.Pos.Line,
			Message:    issue.Text,
		})
	}

	return findings
}

func isLintExit(code int) bool {
	// golangci-lint's pkg/exitcodes reserves 1 (issues found) and 7 (issues surfaced
	// through its own error log) for "ran fine, found something", not a failure.
	return code == 1 || code == 7
}
