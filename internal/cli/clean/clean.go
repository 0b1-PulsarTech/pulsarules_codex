package clean

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/version"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type mode uint8

const (
	report mode = iota
	rewrite
)

func RunClean(inj remy.Injector, opts *version.Options) error {
	root := opts.ProjectDir
	if root == "" {
		root = "."
	}
	// Only --write mutates; the dangerous path is the one a caller has to ask for.
	how := report
	if opts.Write {
		how = rewrite
	}
	repo, err := remy.Get[vcs.Repository](inj)
	if err != nil && !errors.Is(err, vcs.ErrNoRepository) {
		return fmt.Errorf("open repository: %w", err)
	}
	return sweep(os.Stdout, root, how, repo)
}

func sweep(out io.Writer, dir string, how mode, repo vcs.Repository) error {
	root, scope, err := scopeWithin(dir, repo)
	if err != nil {
		return err
	}
	cleaner := cleaner.New(filepath.Join(root, scope))
	var acted, remaining int

	var walkErr error
	walkSources(root, repo, func(rel, _ string) bool {
		if !within(scope, rel) {
			return true
		}
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

func scopeWithin(dir string, repo vcs.Repository) (root, scope string, err error) {
	if repo == nil {
		// The walk starts at dir itself, so the scope is the whole walk.
		return dir, ".", nil
	}
	var absDir, absRoot string
	if absDir, err = resolveDir(dir); err != nil {
		return "", "", err
	}
	if absRoot, err = resolveDir(repo.Root()); err != nil {
		return "", "", err
	}
	if scope, err = filepath.Rel(absRoot, absDir); err != nil || !within(".", scope) {
		return "", "", fmt.Errorf("clean %q: outside the repository at %q", dir, absRoot)
	}
	return repo.Root(), scope, nil
}

func resolveDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", dir, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve %q: %w", dir, err)
	}
	return resolved, nil
}

func within(scope, rel string) bool {
	// Rel compares whole path segments, so scope "sub" does not take "subway/x.md".
	rest, err := filepath.Rel(scope, rel)
	return err == nil && rest != ".." && !strings.HasPrefix(rest, ".."+string(filepath.Separator))
}

func walkSources(root string, repo vcs.Repository, visit func(rel, ext string) bool) {
	skips := source.NewSkipSet(source.DefaultSkipDirs())
	if repo == nil {
		source.NewProvider(root, skips).Walk(visit)
		return
	}
	// The gate's own listing, so clean walks the files the gate walks; the skip set it returns
	// is already applied by the provider.
	provider, _ := governance.ListedSources(
		repo, skips, source.NewSkipSet(source.GitSkipDirs()), slog.Default(),
	)
	provider.Walk(visit)
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
