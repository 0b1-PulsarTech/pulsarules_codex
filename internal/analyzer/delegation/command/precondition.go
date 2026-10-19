package command

import (
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/delegation/command/dartgen"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

const (
	dartPubspec    = "pubspec.yaml"
	dartPackageCfg = ".dart_tool/package_config.json"
	codegenMissing = "uri_has_not_been_generated"
	codeUnresolved = "uri_does_not_exist"
)

func dartPrecheck(root, dir string, listed listing) string {
	for _, pubspecDir := range pubspecDirs(root, dir, listed) {
		if hasPackageConfig(root, pubspecDir) {
			continue
		}
		shown := "."
		if rel, err := filepath.Rel(root, pubspecDir); err == nil {
			shown = filepath.ToSlash(rel)
		}
		return fmt.Sprintf(
			"%s in %s has no %s; run `dart pub get` (then `dart run build_runner build`)",
			dartPubspec, shown, dartPackageCfg,
		)
	}
	return ""
}

func pubspecDirs(root, dir string, listed listing) []string {
	if listed == nil {
		if _, err := os.Stat(filepath.Join(dir, dartPubspec)); err != nil {
			return nil
		}
		return []string{dir}
	}
	prefix := ""
	if rel, err := filepath.Rel(root, dir); err == nil && rel != "." {
		prefix = filepath.ToSlash(rel) + "/"
	}
	var dirs []string
	for file := range listed {
		if path.Base(file) == dartPubspec && strings.HasPrefix(file, prefix) {
			dirs = append(dirs, filepath.Join(root, filepath.FromSlash(path.Dir(file))))
		}
	}
	slices.Sort(dirs)
	return dirs
}

func hasPackageConfig(root, dir string) bool {
	_, found := packageConfigDir(root, dir)
	return found
}

func packageConfigDir(root, dir string) (string, bool) {
	for {
		config := filepath.Join(dir, filepath.FromSlash(dartPackageCfg))
		if stat, err := os.Stat(config); err == nil && stat.Mode().IsRegular() {
			return dir, true
		}
		if dir == root || dir == filepath.Dir(dir) {
			return "", false
		}
		dir = filepath.Dir(dir)
	}
}

func dartCodegen(found []diagnostic, where scope) ([]diagnostic, string) {
	missing := map[string]struct{}{}
	for _, item := range found {
		if target, absent := generatedTarget(where, item); absent {
			missing[target] = struct{}{}
		}
	}
	for _, target := range workspaceMissing(where) {
		missing[target] = struct{}{}
	}
	if len(missing) == 0 {
		return found, ""
	}
	kept := make([]diagnostic, 0, len(found))
	for _, item := range found {
		if item.severity != finding.SeverityError {
			kept = append(kept, item)
		}
	}
	dropped := len(found) - len(kept)
	return kept, fmt.Sprintf(
		"generated code is missing (%s); run `dart run build_runner build`; "+
			"%d derived %s dropped",
		describeMissing(missing), dropped, plural(dropped, "error", "errors"),
	)
}

func generatedTarget(where scope, item diagnostic) (string, bool) {
	uri := item.message
	// The URI is the last quoted span: the text before it holds an apostrophe ("hasn't").
	if end := strings.LastIndex(item.message, "'"); end > 0 {
		if start := strings.LastIndex(item.message[:end], "'"); start >= 0 {
			uri = item.message[start+1 : end]
		}
	}
	switch {
	case item.code == codegenMissing:
	case item.code == codeUnresolved && dartgen.IsGenerated(uri):
	default:
		return "", false
	}
	if rel, inside := relativeTo(where.root, where.dir, item.file); inside {
		if target := dartgen.Resolve(rel, uri); target != "" {
			return target, true
		}
	}
	return uri, true
}

func workspaceMissing(where scope) []string {
	if where.listed == nil {
		return nil
	}
	base, found := packageConfigDir(where.root, where.dir)
	if !found {
		base = where.dir
	}
	prefix := ""
	if rel, err := filepath.Rel(where.root, base); err == nil && rel != "." {
		prefix = filepath.ToSlash(rel) + "/"
	}
	return dartgen.Missing(where.root, prefix, slices.Collect(maps.Keys(where.listed)))
}

func describeMissing(missing map[string]struct{}) string {
	first := slices.Min(slices.Collect(maps.Keys(missing)))
	return fmt.Sprintf("%d %s: %s", len(missing), plural(len(missing), "file", "files"), first)
}

func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}
	return many
}
