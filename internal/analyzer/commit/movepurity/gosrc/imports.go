package gosrc

import (
	"slices"
)

type ImportScope struct {
	OldMod, NewMod Module
	DropOK         []string
	AddOK          []string
	DirMoves       map[string]string
}

func (scope ImportScope) Allowed(oldImports, newImports []ImportSpec) bool {
	remaining := slices.Clone(newImports)
	var oldOnly []ImportSpec
	for _, spec := range oldImports {
		if idx := slices.Index(remaining, spec); idx >= 0 {
			remaining = slices.Delete(remaining, idx, idx+1)
			continue
		}
		oldOnly = append(oldOnly, spec)
	}
	for _, spec := range oldOnly {
		if isLocked(spec) {
			return false
		}
		if idx := scope.rewriteIndex(spec, remaining); idx >= 0 {
			remaining = slices.Delete(remaining, idx, idx+1)
			continue
		}
		if !slices.Contains(scope.DropOK, spec.Path) {
			return false
		}
	}
	for _, spec := range remaining {
		if isLocked(spec) || !slices.Contains(scope.AddOK, spec.Path) {
			return false
		}
	}
	return true
}

func (scope ImportScope) rewriteIndex(spec ImportSpec, candidates []ImportSpec) int {
	oldDir, ok := scope.OldMod.DirOf(spec.Path)
	if !ok {
		return -1
	}
	return slices.IndexFunc(candidates, func(candidate ImportSpec) bool {
		newDir, isUnder := scope.NewMod.DirOf(candidate.Path)
		return isUnder && candidate.Name == spec.Name && scope.DirMoves[oldDir] == newDir &&
			candidate.Path != spec.Path
	})
}

func isLocked(spec ImportSpec) bool {
	return spec.Name == "_" || spec.Name == "." || spec.Path == "C"
}

func LocalName(imports []ImportSpec, importPath, pkg string) string {
	if importPath == "" {
		return ""
	}
	for _, spec := range imports {
		if spec.Path != importPath || isLocked(spec) {
			continue
		}
		if spec.Name == "" {
			return pkg
		}
		return spec.Name
	}
	return ""
}
