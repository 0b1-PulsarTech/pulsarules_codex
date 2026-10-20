package movepurity

import (
	"maps"
	"path"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type nameSide struct {
	mod     gosrc.Module
	snap    *snapshot
	differs map[gosrc.ImportSpec]bool
}

func (moves *moveContext) newBinding(
	oldFile, newFile gosrc.File,
	scope gosrc.ImportScope,
) gosrc.Binding {
	oldSide := nameSide{
		mod: scope.OldMod, snap: moves.head, differs: differing(oldFile.Imports, newFile.Imports),
	}
	newSide := nameSide{
		mod:     scope.NewMod,
		snap:    moves.staged,
		differs: differing(newFile.Imports, oldFile.Imports),
	}
	oldNames := moves.importNames(oldFile.Imports, oldSide)
	newNames := moves.importNames(newFile.Imports, newSide)
	return gosrc.Binding{
		OldPaths:  oldNames.paths,
		NewPaths:  newNames.paths,
		Ambiguous: ambiguousNames(oldNames, newNames),
		Moves:     moves.moveInfos(),
		IsUnknown: oldNames.isUnknown || newNames.isUnknown,
	}
}

func differing(own, other []gosrc.ImportSpec) map[gosrc.ImportSpec]bool {
	remaining := slices.Clone(other)
	diff := map[gosrc.ImportSpec]bool{}
	for _, spec := range own {
		if idx := slices.Index(remaining, spec); idx >= 0 {
			remaining = slices.Delete(remaining, idx, idx+1)
			continue
		}
		diff[spec] = true
	}
	return diff
}

type importedNames struct {
	paths     map[string]string
	clashes   map[string]bool
	isUnknown bool
}

func ambiguousNames(oldNames, newNames importedNames) map[string]bool {
	ambiguous := map[string]bool{}
	maps.Copy(ambiguous, oldNames.clashes)
	maps.Copy(ambiguous, newNames.clashes)
	return ambiguous
}

func (moves *moveContext) importNames(
	imports []gosrc.ImportSpec, at nameSide,
) importedNames {
	names := importedNames{
		paths:   make(map[string]string, len(imports)),
		clashes: map[string]bool{},
	}
	for _, spec := range imports {
		name, isKnown := moves.importedName(spec, at)
		names.isUnknown = names.isUnknown || !isKnown
		if _, taken := names.paths[name]; taken {
			names.clashes[name] = true
			continue
		}
		names.paths[name] = spec.Path
	}
	return names
}

func (moves *moveContext) importedName(spec gosrc.ImportSpec, at nameSide) (string, bool) {
	if spec.Name != "" {
		return spec.Name, true
	}
	if dir, ok := at.mod.DirOf(spec.Path); ok {
		if found := at.snap.packageName(dir); found != "" {
			return found, true
		}
	}
	if found := moves.movedPackage(spec.Path, at); found != "" {
		return found, true
	}
	// The last path element only guesses the package name, which matters once the import differs.
	return path.Base(spec.Path), !at.differs[spec]
}

func (moves *moveContext) movedPackage(importPath string, at nameSide) string {
	for _, info := range moves.moveInfos() {
		if at.snap.side == vcs.Staged && info.NewFull == importPath {
			return info.NewPkg
		}
		if at.snap.side == vcs.Head && info.OldFull == importPath {
			return info.OldPkg
		}
	}
	return ""
}
