package tssyntax

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsalias"
)

type importer struct {
	file, dir, root string
	aliases         []tsalias.Alias
}

func applyPrefer(file *syntax.File, from importer) {
	from.file = filepath.ToSlash(file.Path)
	from.dir = path.Dir(from.file)
	for idx := range file.Imports {
		imported := &file.Imports[idx]
		spec, ok := unquote(imported.Literal)
		if !ok {
			continue
		}
		if isRelative(spec) {
			from.relative(imported, spec)
			continue
		}
		imported.Prefer = from.neighbour(spec)
	}
}

func unquote(literal string) (string, bool) {
	if len(literal) < 2 || !strings.ContainsRune("'\"", rune(literal[0])) ||
		literal[len(literal)-1] != literal[0] {
		return "", false
	}
	return literal[1 : len(literal)-1], true
}

func isRelative(spec string) bool {
	return spec == "." || spec == ".." || strings.HasPrefix(spec, "./") ||
		strings.HasPrefix(spec, "../")
}

func (from importer) relative(imported *syntax.Import, spec string) {
	target := path.Join(from.dir, spec)
	if _, below := tsalias.Within(from.dir, target); below {
		return
	}
	if _, inside := tsalias.Within(from.root, target); from.root != "" && !inside {
		imported.Foreign = true
		return
	}
	imported.Prefer = from.spelling(target)
	imported.Upward = imported.Prefer == ""
}

func (from importer) spelling(target string) string {
	best, bestLen := "", -1
	for _, alias := range from.aliases {
		if !alias.Usable() || !alias.Serves(from.file) {
			continue
		}
		candidate, ok := alias.Spelling(target)
		if ok && len(alias.Target) > bestLen && tsalias.Spellable(target, candidate) {
			best, bestLen = candidate, len(alias.Target)
		}
	}
	return best
}

func (from importer) neighbour(spec string) string {
	alias, ok := tsalias.Best(from.aliases, spec)
	if !ok || !alias.Usable() {
		return ""
	}
	resolved, _ := alias.Resolve(spec)
	if path.Dir(resolved) != from.dir || !tsalias.Spellable(resolved, spec) {
		return ""
	}
	return "./" + path.Base(resolved)
}
