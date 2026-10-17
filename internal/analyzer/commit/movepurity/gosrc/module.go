package gosrc

import (
	"path"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
)

type Module struct {
	path string
	root string
}

func WalkModule(dir string, read func(string) (string, error)) Module {
	for current := dir; ; current = path.Dir(current) {
		if src, err := read(path.Join(current, "go.mod")); err == nil {
			return Module{path: gomod.ModulePath([]byte(src)), root: current}
		}
		if current == "." || current == "/" {
			return Module{}
		}
	}
}

func (mod Module) ImportPath(dir string) string {
	if mod.path == "" {
		return ""
	}
	rel := dir
	if mod.root != "." {
		rel = strings.TrimPrefix(dir, mod.root)
		rel = strings.TrimPrefix(rel, "/")
	}
	if rel == "." || rel == "" {
		return mod.path
	}
	return mod.path + "/" + rel
}

func (mod Module) DirOf(importPath string) (string, bool) {
	rest, ok := strings.CutPrefix(importPath, mod.path)
	if mod.path == "" || !ok || (rest != "" && rest[0] != '/') {
		return "", false
	}
	return path.Join(mod.root, rest), true
}
