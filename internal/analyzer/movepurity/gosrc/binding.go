package gosrc

import "slices"

type Binding struct {
	OldPaths  map[string]string
	NewPaths  map[string]string
	Moves     []MoveInfo
	Captures  CaptureSets
	IsUnknown bool
}

func (binding Binding) allows(qualifier, name string) bool {
	oldPath, newPath := binding.OldPaths[qualifier], binding.NewPaths[qualifier]
	if oldPath == newPath {
		if oldPath == "" {
			return !binding.IsUnknown
		}
		return !binding.movedAway(oldPath, name)
	}
	return slices.ContainsFunc(binding.Moves, func(info MoveInfo) bool {
		return oldPath != "" && info.OldFull == oldPath && info.NewFull == newPath &&
			info.Decls[name]
	})
}

func (binding Binding) movedAway(importPath, name string) bool {
	return slices.ContainsFunc(binding.Moves, func(info MoveInfo) bool {
		return info.OldFull == importPath && info.NewFull != importPath && info.Decls[name]
	})
}

type MoveInfo struct {
	OldDir, NewDir   string
	OldFull, NewFull string
	OldPkg, NewPkg   string
	Decls            map[string]bool
}
