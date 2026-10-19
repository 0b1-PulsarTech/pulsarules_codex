package command

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func (analyzer *Analyzer) runEntry(
	root string, listed listing, entry pass.DelegatedCommand,
) []finding.Finding {
	rep := reports{id: entry.ID, rule: entry.Rule}
	if physical, err := filepath.EvalSymlinks(root); err == nil {
		root = physical
	}
	target, err := analyzer.checkEntry(root, entry)
	if err == nil && !listed.holds(root, target.dir) {
		err = fmt.Errorf(
			"dir %q holds no file governance analyses (ignored, skipped dir, or submodule)",
			entry.Dir,
		)
	}
	if err != nil {
		return []finding.Finding{rep.config(err.Error())}
	}
	if reason := target.kind.precheck(root, target.dir, listed); reason != "" {
		return []finding.Finding{rep.skipped(reason)}
	}
	name := entry.Run[0]
	if _, lookErr := exec.LookPath(resolveInDir(target.dir, name)); lookErr != nil {
		return []finding.Finding{rep.unavailable(name)}
	}
	result, err := Execute(process.Command{
		Name: name,
		Args: entry.Run[1:],
		Dir:  target.dir,
		// A cold dart analyze of an 860-file Flutter app takes about 6s; five minutes leaves
		// room for a far larger project or a slow CI disk without hanging a hook forever.
		Timeout: commandTimeout,
	})
	inv := invocation{rep: rep, kind: target.kind, name: name}
	failure, reported := classify(inv, err, result)
	if failure != nil {
		return []finding.Finding{*failure}
	}
	where := scope{root: root, dir: target.dir, listed: listed}
	return decodeFindings(inv, where, result, reported)
}

type invocation struct {
	rep  reports
	kind format
	name string
}

type Outcome int

const (
	Completed Outcome = iota
	OutputExceeded
	TimedOut
	Reported
	Exited
	NotRun
)

func Classify(err error, reportingExits []int) (Outcome, int) {
	if err == nil {
		return Completed, 0
	}
	var exitErr *exec.ExitError
	switch {
	case errors.Is(err, process.ErrOutputExceeded):
		return OutputExceeded, 0
	case errors.Is(err, context.DeadlineExceeded):
		return TimedOut, 0
	case errors.As(err, &exitErr) && slices.Contains(reportingExits, exitErr.ExitCode()):
		return Reported, exitErr.ExitCode()
	case exitErr != nil:
		return Exited, exitErr.ExitCode()
	}
	return NotRun, 0
}

func Execute(cmd process.Command) (process.Result, error) {
	if cmd.MaxOutput == 0 {
		cmd.MaxOutput = MaxOutputBytes // per stream; a delegated tool once printed 13.8 MB
	}
	return process.Run(context.Background(), cmd)
}

func classify(inv invocation, err error, result process.Result) (*finding.Finding, int) {
	outcome, code := Classify(err, inv.kind.reportingExits)
	var failure finding.Finding
	switch outcome {
	case Completed, Reported:
		return nil, code
	case OutputExceeded:
		failure = inv.rep.failed(
			fmt.Sprintf("output exceeded %d bytes", MaxOutputBytes), result.Stderr,
		)
	case TimedOut:
		failure = inv.rep.timeout(inv.name)
	case Exited:
		failure = inv.rep.failed(
			fmt.Sprintf("%s exited with code %d", inv.name, code), result.Stderr,
		)
	case NotRun:
		failure = inv.rep.failed(
			fmt.Sprintf("%s did not run: %v", inv.name, cause(err)), result.Stderr,
		)
	}
	return &failure, 0
}

func cause(err error) error {
	if execErr, found := errors.AsType[*process.Error](err); found {
		return execErr.Err
	}
	return err
}

type scope struct {
	root   string
	dir    string
	listed listing
}

type placed struct {
	rel     string
	outcome verdict
}

func (where scope) place(file string) placed {
	if file == "" {
		return placed{outcome: kept}
	}
	rel, inside := relativeTo(where.root, where.dir, file)
	if !inside {
		return placed{outcome: dropped}
	}
	return placed{rel: rel, outcome: where.listed.judge(where.root, where.dir, rel)}
}

func decodeFindings(
	inv invocation, where scope, result process.Result, reported int,
) []finding.Finding {
	found, err := inv.kind.decode([]byte(result.Stdout))
	if err != nil {
		return []finding.Finding{inv.rep.failed("undecodable output: "+err.Error(), result.Stderr)}
	}
	found, notice := inv.kind.codegen(found, where)
	sorted := where.sortOut(inv.rep, found)
	if notice != "" {
		sorted.findings = append(sorted.findings, inv.rep.notice(notice))
	} else if reported != 0 && len(sorted.findings) == 0 && sorted.excluded == 0 {
		reason := fmt.Sprintf("exit %d but no diagnostics inside the project", reported)
		return []finding.Finding{inv.rep.failed(reason, result.Stderr)}
	}
	if sorted.lost.count > 0 {
		return append(sorted.findings, inv.rep.dropped(sorted.lost))
	}
	return sorted.findings
}
