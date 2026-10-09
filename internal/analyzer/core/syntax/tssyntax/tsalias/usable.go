package tsalias

import (
	"path"
	"slices"
	"strings"
)

func (alias Alias) Usable() bool {
	return !isDeclaration(alias.Target) &&
		!slices.Contains(strings.Split(alias.Target, "/"), "node_modules")
}

func (alias Alias) Serves(importer string) bool {
	switch alias.Origin {
	case OriginKit:
		_, below := Within(path.Join(alias.Root, "src"), importer)
		return below
	case OriginPaths:
		return !slices.Contains([]string{".js", ".mjs", ".cjs"}, path.Ext(importer))
	}
	return true
}

func Best(aliases []Alias, spec string) (Alias, bool) {
	const exact = 1 << 30
	var best Alias
	bestScore := -1
	for _, alias := range aliases {
		if _, ok := Match(alias.Spec, spec); !ok {
			continue
		}
		score := exact
		if prefix, _, wild := strings.Cut(alias.Spec, "*"); wild {
			score = len(prefix)
		}
		if score > bestScore {
			best, bestScore = alias, score
		}
	}
	return best, bestScore >= 0
}

func Spellable(resolved, spec string) bool {
	if isDeclaration(resolved) {
		return false
	}
	ext := path.Ext(resolved)
	switch ext {
	case ".ts", ".tsx", ".mts", ".cts":
		return strings.HasSuffix(spec, ext)
	}
	return true
}

func isDeclaration(file string) bool {
	return strings.HasSuffix(file, ".d.ts") || strings.HasSuffix(file, ".d.mts") ||
		strings.HasSuffix(file, ".d.cts")
}
