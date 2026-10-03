package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/cliopts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/clean"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/mark"
)

type mode uint8

const (
	report mode = iota
	rewrite
)

func runClean(_ remy.Injector, opts *cliopts.Options) error {
	root := opts.ProjectDir
	if root == "" {
		root = "."
	}
	// Only --write mutates; the dangerous path is the one a caller has to ask for.
	how := report
	if opts.Write {
		how = rewrite
	}
	return sweep(os.Stdout, root, how)
}

func sweep(out io.Writer, root string, how mode) error {
	cleaner := clean.New(root)
	var acted, remaining int

	var walkErr error
	sources.NewProvider(root, sources.NewSkipSet(sources.DefaultSkipDirs())).
		Walk(func(rel, _ string) bool {
			// Walk yields a path relative to root; Abs would anchor it to the working directory.
			found, err := inspectOrClean(cleaner, filepath.Join(root, rel), how)
			if err != nil {
				walkErr = err
				return false
			}
			if found.Skipped() {
				return true
			}
			acted += len(found.Acted)
			remaining += len(found.Remaining)
			printReport(out, found, how)
			return true
		})
	if walkErr != nil {
		return fmt.Errorf("clean %q: %w", root, walkErr)
	}

	if how == rewrite {
		_, _ = fmt.Fprintf(
			out,
			"removed %d carrier(s); %d marker(s) left for a human\n",
			acted,
			remaining,
		)
		return nil
	}
	_, _ = fmt.Fprintf(
		out,
		"%d marker(s) found; re-run with --write to remove the removable ones\n",
		remaining,
	)
	return nil
}

func inspectOrClean(cleaner *clean.Cleaner, path string, how mode) (clean.Report, error) {
	if how == rewrite {
		return cleaner.CleanFile(path)
	}
	return cleaner.Inspect(path)
}

func printReport(out io.Writer, found clean.Report, how mode) {
	for _, acted := range found.Acted {
		_, _ = fmt.Fprintf(out, "%s:%d removed %s\n", found.Path, acted.Line, acted.Name)
	}
	verb := "found"
	if how == rewrite {
		verb = "left"
	}
	for _, left := range found.Remaining {
		_, _ = fmt.Fprintf(
			out, "%s:%d %s %s (%s)\n", found.Path, left.Line, verb, left.Name, advice(left.Class),
		)
	}
}

func advice(class mark.Class) string {
	switch class {
	case mark.ClassTypographic:
		return "replace with its ASCII form by hand"
	case mark.ClassContextual:
		return "may be load-bearing; judge it before removing"
	case mark.ClassStrip, mark.ClassSpace:
		return "removable"
	}
	return "unclassified"
}
