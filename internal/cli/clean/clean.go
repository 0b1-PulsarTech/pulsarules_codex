package clean

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/discovery"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

func sweep(out io.Writer, dir string, how sweepStrategy, repo discovery.Lister) error {
	root, scope, err := scopeWithin(dir, repo)
	if err != nil {
		return err
	}
	scrub := cleaner.New(filepath.Join(root, scope))
	var tally cleaner.Tally
	var walkErr error
	walkSources(root, repo, func(rel, _ string) bool {
		if !within(scope, rel) {
			return true
		}
		// Walk yields a path relative to root; Abs would anchor it to the working directory.
		found, err := how.apply(scrub, filepath.Join(root, rel))
		if err != nil {
			walkErr = err
			return false
		}
		if found.Skipped() && len(found.Remaining) == 0 {
			return true
		}
		tally.Add(found)
		printReport(out, found, how)
		return true
	})
	if walkErr != nil {
		return fmt.Errorf("clean %q: %w", root, walkErr)
	}

	_, _ = fmt.Fprintln(out, how.summary(tally))
	return nil
}

func scopeWithin(dir string, repo discovery.Lister) (root, scope string, err error) {
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

func walkSources(root string, repo discovery.Lister, visit func(rel, ext string) bool) {
	skips := source.NewSkipSet(source.DefaultSkipDirs())
	if repo == nil {
		source.NewProvider(root, skips).Walk(visit)
		return
	}
	// The gate's own listing, so clean walks the files the gate walks; the skip set it returns
	// is already applied by the provider.
	provider, _ := discovery.Listed(
		repo, skips, source.NewSkipSet(source.GitSkipDirs()), slog.Default(),
	)
	provider.Walk(visit)
}

func printReport(out io.Writer, found cleaner.Report, how sweepStrategy) {
	path := printable.Path(found.Path)
	if found.Skipped() {
		_, _ = fmt.Fprintf(out, "%s: %s\n", path, found.SkipNote())
		return
	}
	for _, acted := range found.Acted {
		_, _ = fmt.Fprintf(out, "%s:%d removed %s\n", path, acted.Line, acted.Name)
	}
	for _, left := range found.Remaining {
		note := advice(left.Class, filepath.Ext(found.Path), found.EscapeAdvice(left))
		if held := found.HeldNote(left); held != "" {
			note = held
		}
		_, _ = fmt.Fprintf(out, "%s:%d %s %s (%s)\n", path, left.Line, how.verb, left.Name, note)
	}
}

func advice(class invisible.Class, ext, escape string) string {
	caveat := "; inside a string literal it is data: " + escape
	removable := "removable outside a string literal; inside one, " + escape
	if markdown.IsExt(ext) {
		caveat, removable = "", "removable outside possible code"
	}
	switch class {
	case invisible.ClassTypographic:
		return "replace with its ASCII form by hand" + caveat
	case invisible.ClassContextual:
		return "may be load-bearing; judge it before removing"
	case invisible.ClassStrip, invisible.ClassSpace:
		return removable
	}
	return "unclassified"
}
