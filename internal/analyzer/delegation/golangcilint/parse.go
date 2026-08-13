package golangcilint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
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

	findings, escaped := issueFindings(result.Issues)
	// A stale cache is the environment's problem, not the code's, so it
	// warns instead of failing the gate the way a real lint finding does.
	if escaped > 0 {
		findings = append(findings, r.escapedWarning(escaped))
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

func issueFindings(issues []lintIssue) (findings []core.Finding, escaped int) {
	for _, issue := range issues {
		if escapesProject(issue.Pos.Filename) {
			escaped++
			continue
		}
		severity := core.SeverityWarning
		if strings.Contains(issue.Severity, "error") {
			severity = core.SeverityError
		}
		findings = append(findings, core.Finding{
			AnalyzerID: fmt.Sprintf("golangci-lint/%s", issue.FromLinter),
			Severity:   severity,
			Category:   core.CatSyntax,
			File:       issue.Pos.Filename,
			Line:       issue.Pos.Line,
			Message:    issue.Text,
		})
	}
	return findings, escaped
}

func (r *Runner) escapedWarning(escaped int) core.Finding {
	return r.warning.NewWithSuggestion(
		fmt.Sprintf("dropped %d finding(s) naming files outside the analysed project", escaped),
		"run `golangci-lint cache clean`; a stale cache reports paths from a directory that is "+
			"gone",
	)
}

func escapesProject(path string) bool {
	// A path that climbs out of cmd.Dir comes from a stale cache keyed to a missing directory.
	return strings.HasPrefix(filepath.ToSlash(filepath.Clean(path)), "../")
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
