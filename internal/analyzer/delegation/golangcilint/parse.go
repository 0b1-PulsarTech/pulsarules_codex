package golangcilint

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

var staleCacheReporter = core.NewReporter(
	"golangci-lint", core.SeverityWarning, core.CatSyntax,
)

func parseOutput(result execx.Result, cmdErr error) []core.Finding {
	if cmdErr != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(cmdErr, &exitErr):
			if !isLintExit(exitErr.ExitCode()) {
				return []core.Finding{
					golangciLintReporter.New(
						fmt.Sprintf("golangci-lint failed: %s", result.Stderr),
					),
				}
			}
		default:
			// why: a non-ExitError (e.g. a start failure, or the process
			// killed on timeout) means the command never produced stdout to
			// parse; report the real cause instead of falling through to a
			// JSON parse failure on empty output.
			return []core.Finding{
				golangciLintReporter.New(fmt.Sprintf("golangci-lint failed: %s", cmdErr)),
			}
		}
	}

	jsonOut := extractJSON([]byte(result.Stdout))

	var parsed struct {
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

	if parseErr := json.Unmarshal(jsonOut, &parsed); parseErr != nil {
		return []core.Finding{
			golangciLintReporter.New(
				fmt.Sprintf("failed to parse golangci-lint output: %s", parseErr),
			),
		}
	}

	var findings []core.Finding
	var escaped int
	for _, issue := range parsed.Issues {
		if escapesProject(issue.Pos.Filename) {
			escaped++
			continue
		}
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

	if escaped > 0 {
		// why: a stale cache is the environment's problem, not the code's, so it
		// warns instead of failing the gate the way a real lint finding does.
		findings = append(findings, staleCacheReporter.NewWithSuggestion(
			fmt.Sprintf("dropped %d finding(s) naming files outside the analysed project", escaped),
			"run `golangci-lint cache clean`; a stale cache reports paths from a directory that is gone",
		))
	}
	return findings
}

// why: golangci-lint is run with cmd.Dir set to the project, so every honest
// finding is inside it. A path that climbs out comes from a stale cache keyed to
// a directory that no longer exists, and printing it as real sent four separate
// measurements in this repo chasing files nobody could open.
func escapesProject(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(filepath.Clean(path)), "../")
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
