package clean

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type mode uint8

const (
	report mode = iota
	rewrite
)

func RunClean(_ remy.Injector, opts *version.Options) error {
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
	cleaner := cleaner.New(root)
	var acted, remaining int

	var walkErr error
	source.NewProvider(root, source.NewSkipSet(source.DefaultSkipDirs())).
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

func inspectOrClean(cleaner *cleaner.Cleaner, path string, how mode) (cleaner.Report, error) {
	if how == rewrite {
		return cleaner.CleanFile(path)
	}
	return cleaner.Inspect(path)
}

func printReport(out io.Writer, found cleaner.Report, how mode) {
	path := printable.Path(found.Path)
	for _, acted := range found.Acted {
		_, _ = fmt.Fprintf(out, "%s:%d removed %s\n", path, acted.Line, acted.Name)
	}
	verb := "found"
	if how == rewrite {
		verb = "left"
	}
	for _, left := range found.Remaining {
		_, _ = fmt.Fprintf(
			out, "%s:%d %s %s (%s)\n", path, left.Line, verb, left.Name, advice(left.Class),
		)
	}
}

func advice(class invisible.Class) string {
	switch class {
	case invisible.ClassTypographic:
		return "replace with its ASCII form by hand"
	case invisible.ClassContextual:
		return "may be load-bearing; judge it before removing"
	case invisible.ClassStrip, invisible.ClassSpace:
		return "removable"
	}
	return "unclassified"
}
