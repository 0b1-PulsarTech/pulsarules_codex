package tssyntax

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/pathalias"
)

type importSite struct {
	file, dir, root string
	aliases         []pathalias.Alias
}

func newImportSite(file, root string, aliases []pathalias.Alias) importSite {
	slashed := filepath.ToSlash(file)
	return importSite{file: slashed, dir: path.Dir(slashed), root: root, aliases: aliases}
}

func applyPrefer(file *syntax.File, from importSite) {
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

func (from importSite) relative(imported *syntax.Import, spec string) {
	target := path.Join(from.dir, spec)
	if _, below := pathalias.Within(from.dir, target); below {
		return
	}
	if _, inside := pathalias.Within(from.root, target); from.root != "" && !inside {
		imported.Foreign = true
		return
	}
	imported.Prefer = from.spelling(target)
	imported.Upward = imported.Prefer == ""
}

func (from importSite) spelling(target string) string {
	best, bestLen := "", -1
	for _, alias := range from.aliases {
		if !alias.Usable() || !alias.Serves(from.file) {
			continue
		}
		candidate, ok := alias.Spelling(target)
		if ok && len(alias.Target) > bestLen && pathalias.Spellable(target, candidate) {
			best, bestLen = candidate, len(alias.Target)
		}
	}
	return best
}

func (from importSite) neighbour(spec string) string {
	alias, ok := pathalias.Best(from.aliases, spec)
	if !ok || !alias.Usable() {
		return ""
	}
	resolved, _ := alias.Resolve(spec)
	if path.Dir(resolved) != from.dir || !pathalias.Spellable(resolved, spec) {
		return ""
	}
	return "./" + path.Base(resolved)
}
