package gosrc

import (
	"path"
	"strconv"
	"strings"
)

type Module struct {
	path string
	root string
}

func WalkModule(dir string, read func(string) (string, error)) Module {
	for current := dir; ; current = path.Dir(current) {
		if src, err := read(path.Join(current, "go.mod")); err == nil {
			return Module{path: modulePath(src), root: current}
		}
		if current == "." || current == "/" {
			return Module{}
		}
	}
}

func modulePath(goMod string) string {
	for line := range strings.SplitSeq(goMod, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			if unquoted, err := strconv.Unquote(fields[1]); err == nil {
				return unquoted
			}
			return fields[1]
		}
	}
	return ""
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
