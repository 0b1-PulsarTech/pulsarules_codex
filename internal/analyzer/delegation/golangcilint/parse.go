package golangcilint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
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

func (r *Runner) parseOutput(out []byte, cmdErr error) []core.Finding {
	if failure := r.exitFailure(cmdErr); failure != nil {
		return failure
	}

	var result lintReport
	if parseErr := json.Unmarshal(extractJSON(out), &result); parseErr != nil {
		return []core.Finding{
			r.failure.New(
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

func (r *Runner) exitFailure(cmdErr error) []core.Finding {
	if cmdErr == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(cmdErr, &exitErr) {
		// A non-ExitError means the command never produced stdout to parse;
		// report the real cause, not a JSON parse failure on empty output.
		return []core.Finding{
			r.failure.New(fmt.Sprintf("golangci-lint failed: %s", cmdErr)),
		}
	}
	if isLintExit(exitErr.ExitCode()) {
		return nil
	}
	return []core.Finding{
		r.failure.New(fmt.Sprintf("golangci-lint failed: %s", string(exitErr.Stderr))),
	}
}

func isLintExit(code int) bool {
	const (
		lintExitIssuesFound    = 1
		lintExitErrorWasLogged = 7
	)
	// golangci-lint's pkg/exitcodes reserves 1 (issues found) and 7 (issues surfaced
	// through its own error log) for "ran fine, found something", not a failure.
	return code == lintExitIssuesFound || code == lintExitErrorWasLogged
}
