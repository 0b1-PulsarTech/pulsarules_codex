package command

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/execx"
)

func (analyzer *Analyzer) runEntry(
	root string, listed listing, entry core.DelegatedCommand,
) []core.Finding {
	rep := reports{id: entry.ID, rule: entry.Rule}
	if physical, err := filepath.EvalSymlinks(root); err == nil {
		root = physical
	}
	kind, dir, fault := analyzer.checkEntry(root, entry)
	if fault == "" && !listed.holds(root, dir) {
		fault = fmt.Sprintf(
			"dir %q holds no file governance analyses (ignored, skipped dir, or submodule)",
			entry.Dir,
		)
	}
	if fault != "" {
		return []core.Finding{rep.config(fault)}
	}
	if kind.precheck != nil {
		if reason := kind.precheck(root, dir, listed); reason != "" {
			return []core.Finding{rep.skipped(reason)}
		}
	}
	name := entry.Run[0]
	if _, err := exec.LookPath(resolveInDir(dir, name)); err != nil {
		return []core.Finding{rep.unavailable(name)}
	}
	result, err := execx.Run(context.Background(), execx.Command{
		Name: name,
		Args: entry.Run[1:],
		Dir:  dir,
		// A cold dart analyze of the 860-file terectek_app takes about 6s; five minutes leaves
		// room for a far larger project or a slow CI disk without hanging a hook forever.
		Timeout: commandTimeout,
		// Per stream. A real run is 0.3 MB; a project with no pub get reported 24937 diagnostics
		// in 13.8 MB, so 32 MiB admits that case and still bounds a runaway tool.
		MaxOutput: maxOutputBytes,
	})
	inv := invocation{rep: rep, kind: kind, name: name}
	failure, reported := classify(inv, err, result)
	if failure != nil {
		return []core.Finding{*failure}
	}
	where := scope{root: root, dir: dir, listed: listed}
	return decodeFindings(inv, where, result, reported)
}

type invocation struct {
	rep  reports
	kind format
	name string
}

func classify(inv invocation, err error, result execx.Result) (*core.Finding, int) {
	if err == nil {
		return nil, 0
	}
	var failure core.Finding
	var exitErr *exec.ExitError
	switch {
	case errors.Is(err, execx.ErrOutputExceeded):
		failure = inv.rep.failed(
			fmt.Sprintf("output exceeded %d bytes", maxOutputBytes), result.Stderr,
		)
	case errors.Is(err, context.DeadlineExceeded):
		failure = inv.rep.timeout(inv.name)
	case errors.As(err, &exitErr) && slices.Contains(inv.kind.reportingExits, exitErr.ExitCode()):
		return nil, exitErr.ExitCode()
	case exitErr != nil:
		failure = inv.rep.failed(
			fmt.Sprintf("%s exited with code %d", inv.name, exitErr.ExitCode()), result.Stderr,
		)
	default:
		failure = inv.rep.failed(
			fmt.Sprintf("%s did not run: %v", inv.name, cause(err)), result.Stderr,
		)
	}
	return &failure, 0
}

func cause(err error) error {
	if execErr, found := errors.AsType[*execx.Error](err); found {
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
	inv invocation, where scope, result execx.Result, reported int,
) []core.Finding {
	found, err := inv.kind.decode([]byte(result.Stdout))
	if err != nil {
		return []core.Finding{inv.rep.failed("undecodable output: "+err.Error(), result.Stderr)}
	}
	sorted := where.sortOut(inv.rep, found)
	if reported != 0 && len(sorted.findings) == 0 && sorted.excluded == 0 {
		reason := fmt.Sprintf("exit %d but no diagnostics inside the project", reported)
		return []core.Finding{inv.rep.failed(reason, result.Stderr)}
	}
	if sorted.lost.count > 0 {
		return append(sorted.findings, inv.rep.dropped(sorted.lost))
	}
	return sorted.findings
}
