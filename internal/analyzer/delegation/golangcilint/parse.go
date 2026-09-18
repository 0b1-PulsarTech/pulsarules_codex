package golangcilint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func unavailable(stderr string) bool {
	const lockContention = "parallel golangci-lint is running"
	// Blocking the gate on lock contention fails a commit for something outside
	// the author's code, which is how a gate stops being run at all.
	return strings.Contains(stderr, lockContention)
}

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

func (r *Runner) parseOutput(result process.Result, cmdErr error) []pass.Finding {
	if failure := r.exitFailure(result, cmdErr); failure != nil {
		return failure
	}

	var parsed lintReport
	if parseErr := json.Unmarshal(extractJSON([]byte(result.Stdout)), &parsed); parseErr != nil {
		return []pass.Finding{
			r.failure.New(
				fmt.Sprintf("failed to parse golangci-lint output: %s", parseErr),
			),
		}
	}

	findings, escaped := issueFindings(parsed.Issues)
	// A stale cache is the environment's problem, not the code's, so it
	// warns instead of failing the gate the way a real lint finding does.
	if escaped > 0 {
		findings = append(findings, r.escapedWarning(escaped))
	}
	return findings
}

func (r *Runner) exitFailure(result process.Result, cmdErr error) []pass.Finding {
	if cmdErr == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if !errors.As(cmdErr, &exitErr) {
		// A non-ExitError means the command never produced stdout to parse;
		// report the real cause, not a JSON parse failure on empty output.
		return []pass.Finding{
			r.failure.New(fmt.Sprintf("golangci-lint failed: %s", cmdErr)),
		}
	}
	if isLintExit(exitErr.ExitCode()) {
		return nil
	}
	if unavailable(result.Stderr) {
		return []pass.Finding{r.warning.New(
			"golangci-lint was busy elsewhere; its checks did not run",
		)}
	}
	return []pass.Finding{
		r.failure.New(fmt.Sprintf("golangci-lint failed: %s", result.Stderr)),
	}
}

func issueFindings(issues []lintIssue) (findings []pass.Finding, escaped int) {
	for _, issue := range issues {
		if escapesProject(issue.Pos.Filename) {
			escaped++
			continue
		}
		severity := pass.SeverityWarning
		if strings.Contains(issue.Severity, "error") {
			severity = pass.SeverityError
		}
		findings = append(findings, pass.Finding{
			AnalyzerID: fmt.Sprintf("golangci-lint/%s", issue.FromLinter),
			Severity:   severity,
			Category:   pass.CatSyntax,
			File:       issue.Pos.Filename,
			Line:       issue.Pos.Line,
			Message:    issue.Text,
		})
	}
	return findings, escaped
}

func (r *Runner) escapedWarning(escaped int) pass.Finding {
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
