package detect

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

const DefaultMaxDepth = 3

func scan(root string, maxDepth int) (Set, error) {
	skip := source.SkipSet{}
	var found []Language
	err := fs.WalkDir(os.DirFS(root), ".", func(
		path string, entry fs.DirEntry, walkErr error,
	) error {
		if walkErr != nil {
			// A failure at the root means the project itself is unreadable, which
			// must not be reported as "no languages found"; deeper, one bad entry
			// is skipped so a single unreadable subdir cannot blind detection.
			if path == "." {
				return walkErr
			}
			return nil //nolint:nilerr // skip the bad entry, keep walking.
		}
		if entry.IsDir() {
			if path == "." {
				return nil
			}
			// skip reuses the analyzers' default skip dirs (so a vendored
			// pubspec.yaml under node_modules never signals Dart/JS); maxDepth
			// counts path separators from root, so apps/nucleon/pubspec.yaml is depth 2.
			if skip.SkipsDir(entry.Name()) || strings.Count(path, "/") >= maxDepth {
				return fs.SkipDir
			}
			return nil
		}
		if lang, ok := markerLanguage(entry.Name()); ok {
			found = append(found, lang)
		}
		return nil
	})
	if err != nil {
		return Set{}, fmt.Errorf("walk %q: %w", root, err)
	}
	return NewSet(found...), nil
}

func markerLanguage(name string) (Language, bool) {
	// A language may have more than one marker (go.mod, go.work).
	switch name {
	case "go.mod", "go.work":
		return Go, true
	case "pubspec.yaml":
		return Dart, true
	case "package.json":
		return JavaScript, true
	case "tsconfig.json":
		return TypeScript, true
	case "svelte.config.js":
		return Svelte, true
	case "Cargo.toml":
		return Rust, true
	case "pyproject.toml":
		return Python, true
	}
	return "", false
}
