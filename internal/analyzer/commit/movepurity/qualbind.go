package movepurity

import (
	"path"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
)

type nameSide struct {
	mod      gosrc.Module
	isStaged bool
	differs  map[gosrc.ImportSpec]bool
}

func (moves *moveContext) newBinding(
	oldFile, newFile gosrc.File,
	scope gosrc.ImportScope,
) gosrc.Binding {
	oldSide := nameSide{mod: scope.OldMod, differs: differing(oldFile.Imports, newFile.Imports)}
	newSide := nameSide{
		mod: scope.NewMod, isStaged: true, differs: differing(newFile.Imports, oldFile.Imports),
	}
	oldPaths, isOldUnknown := moves.importNames(oldFile.Imports, oldSide)
	newPaths, isNewUnknown := moves.importNames(newFile.Imports, newSide)
	return gosrc.Binding{
		OldPaths:  oldPaths,
		NewPaths:  newPaths,
		Moves:     moves.moveInfos(),
		IsUnknown: isOldUnknown || isNewUnknown,
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

func (moves *moveContext) importNames(
	imports []gosrc.ImportSpec, at nameSide,
) (names map[string]string, isUnknown bool) {
	names = make(map[string]string, len(imports))
	clash := "\x00old"
	if at.isStaged {
		clash = "\x00new"
	}
	for _, spec := range imports {
		name, isKnown := moves.importedName(spec, at)
		isUnknown = isUnknown || !isKnown
		if _, taken := names[name]; taken {
			names[name] = clash
			continue
		}
		names[name] = spec.Path
	}
	return names, isUnknown
}

func (moves *moveContext) importedName(spec gosrc.ImportSpec, at nameSide) (string, bool) {
	if spec.Name != "" {
		return spec.Name, true
	}
	if dir, ok := at.mod.DirOf(spec.Path); ok {
		if found := moves.packageName(moduleKey{isStaged: at.isStaged, dir: dir}); found != "" {
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
		if at.isStaged && info.NewFull == importPath {
			return info.NewPkg
		}
		if !at.isStaged && info.OldFull == importPath {
			return info.OldPkg
		}
	}
	return ""
}
