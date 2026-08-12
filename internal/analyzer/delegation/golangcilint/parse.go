package golangcilint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func parseOutput(out []byte, cmdErr error) []core.Finding {
	if cmdErr != nil {
		var exitErr *exec.ExitError
		if errors.As(cmdErr, &exitErr) {
			if !isLintExit(exitErr.ExitCode()) {
				return []core.Finding{
					golangciLintReporter.New(
						fmt.Sprintf("golangci-lint failed: %s", string(exitErr.Stderr)),
					),
				}
			}
		}
	}

	jsonOut := extractJSON(out)

	var result struct {
		Issues []struct {
			FromLinter string `json:"FromLinter"`
			Text       string `json:"Text"`
			Severity   string `json:"Severity"`
			Pos        struct {
				Filename string `json:"Filename"`
				Line     int    `json:"Line"`
			} `json:"Pos"`
		} `json:"Issues"`
	}

	if parseErr := json.Unmarshal(jsonOut, &result); parseErr != nil {
		return []core.Finding{
			golangciLintReporter.New(
				fmt.Sprintf("failed to parse golangci-lint output: %s", parseErr),
			),
		}
	}

	var findings []core.Finding
	for _, issue := range result.Issues {
		sev := core.SeverityWarning
		if strings.Contains(issue.Severity, "error") {
			sev = core.SeverityError
		}
		findings = append(findings, core.Finding{
			AnalyzerID: fmt.Sprintf("golangci-lint/%s", issue.FromLinter),
			Severity:   sev,
			Category:   core.CatSyntax,
			File:       issue.Pos.Filename,
			Line:       issue.Pos.Line,
			Message:    issue.Text,
		})
	}

	return findings
}

// golangci-lint's own pkg/exitcodes reserves these two codes for "the tool
// ran fine and found something to report", as opposed to a genuine failure
// (config error, timeout, panic): 1 is issues found, 7 is issues found but
// only surfaced through its own error log.
const (
	lintExitIssuesFound    = 1
	lintExitErrorWasLogged = 7
)

func isLintExit(code int) bool {
	return code == lintExitIssuesFound || code == lintExitErrorWasLogged
}
