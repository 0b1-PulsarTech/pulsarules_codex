package golangcilint

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

const maxMessageBytes = 2048

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

func (runner *Runner) parseOutput(result process.Result, cmdErr error) []finding.Finding {
	if failure := runner.exitFailure(result, cmdErr); failure != nil {
		return failure
	}

	var parsed lintReport
	if parseErr := json.Unmarshal(extractJSON([]byte(result.Stdout)), &parsed); parseErr != nil {
		return []finding.Finding{
			runner.failure.New(
				fmt.Sprintf("failed to parse golangci-lint output: %s", parseErr),
			),
		}
	}

	findings, escaped := issueFindings(parsed.Issues)
	// A stale cache is the environment's problem, not the code's, so it
	// warns instead of failing the gate the way a real lint finding does.
	if escaped > 0 {
		findings = append(findings, runner.escapedWarning(escaped))
	}
	return findings
}

func (runner *Runner) exitFailure(result process.Result, cmdErr error) []finding.Finding {
	const (
		issuesFound    = 1
		errorWasLogged = 7 // pkg/exitcodes: "ran fine, found something", not a failure
	)
	outcome, _ := command.Classify(cmdErr, []int{issuesFound, errorWasLogged})
	switch outcome {
	case command.Completed, command.Reported:
		return nil
	case command.Exited:
		if unavailable(result.Stderr) {
			return []finding.Finding{runner.warning.New(
				"golangci-lint was busy elsewhere; its checks did not run",
			)}
		}
		return []finding.Finding{runner.failure.New(
			"golangci-lint failed: " + printable.Tail(result.Stderr, maxMessageBytes),
		)}
	case command.OutputExceeded:
		return []finding.Finding{runner.failure.New(
			"golangci-lint failed: output exceeded its size limit: " +
				printable.Tail(result.Stderr, maxMessageBytes),
		)}
	}
	// Otherwise the command never produced stdout to parse; report the real
	// cause, not a JSON parse failure on empty output.
	return []finding.Finding{runner.failure.New(
		"golangci-lint failed: " + printable.Head(cmdErr.Error(), maxMessageBytes),
	)}
}

func issueFindings(issues []lintIssue) (findings []finding.Finding, escaped int) {
	for _, issue := range issues {
		if escapesProject(issue.Pos.Filename) {
			escaped++
			continue
		}
		severity := finding.SeverityWarning
		if strings.Contains(issue.Severity, "error") {
			severity = finding.SeverityError
		}
		findings = append(findings, finding.Finding{
			AnalyzerID: fmt.Sprintf("golangci-lint/%s", issue.FromLinter),
			Severity:   severity,
			Category:   finding.CatSyntax,
			File:       issue.Pos.Filename,
			Line:       issue.Pos.Line,
			Message:    issue.Text,
		})
	}
	return findings, escaped
}

func (runner *Runner) escapedWarning(escaped int) finding.Finding {
	return runner.warning.NewWithSuggestion(
		fmt.Sprintf("dropped %d finding(s) naming files outside the analysed project", escaped),
		"run `golangci-lint cache clean`; a stale cache reports paths from "+
			"a directory that is gone",
	)
}

func escapesProject(path string) bool {
	// A path that climbs out of cmd.Dir comes from a stale cache keyed to a missing directory.
	_, inside := command.RelativeTo(".", ".", path)
	return !filepath.IsAbs(path) && !inside
}
