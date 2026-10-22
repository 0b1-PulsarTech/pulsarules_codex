package movepurity

import (
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type declScope struct {
	dirPair
	isTest bool
}

func (moves *moveContext) captureSets(
	oldFile, newFile gosrc.File,
	edit fileEdit,
) gosrc.CaptureSets {
	scope := newDeclScope(edit)
	return gosrc.CaptureSets{
		Left:  moves.leftBehind(scope, oldFile.Pkg),
		Taken: moves.takenInNewDir(scope, newFile.Pkg),
	}
}

func newDeclScope(edit fileEdit) declScope {
	return declScope{dirPair: edit.dirs(), isTest: isTestFile(edit.newPath)}
}

func (scope declScope) includes(file string) bool {
	return isProductionGo(file) || (scope.isTest && isGoFile(file))
}

func (moves *moveContext) leftBehind(scope declScope, pkg string) map[string]bool {
	return cached(moves.left, scopeKey{scope: scope, pkg: pkg}, func() map[string]bool {
		set := map[string]bool{}
		files, err := moves.head.dir(scope.oldDir)
		if err != nil {
			return set
		}
		for _, file := range files {
			// A file co-moved into newDir takes its names along; one moved elsewhere
			// still bound them at HEAD.
			dest, isMoved := moves.movedTo[file]
			if !scope.includes(file) || (isMoved && path.Dir(dest) == scope.newDir) {
				continue
			}
			moves.head.addDecls(file, pkg, set)
		}
		return set
	})
}

func (moves *moveContext) takenInNewDir(scope declScope, pkg string) map[string]bool {
	return cached(moves.taken, scopeKey{scope: scope, pkg: pkg}, func() map[string]bool {
		set := map[string]bool{}
		files, err := moves.staged.dir(scope.newDir)
		if err != nil {
			return set
		}
		for _, file := range files {
			if scope.includes(file) && !moves.isComovedFrom(file, scope.oldDir) {
				moves.staged.addDecls(file, pkg, set)
			}
		}
		return set
	})
}

func (moves *moveContext) isComovedFrom(newPath, oldDir string) bool {
	for _, rename := range moves.renames {
		if rename.NewPath == newPath && path.Dir(rename.OldPath) == oldDir {
			return true
		}
	}
	return false
}

func (moves *moveContext) pathFollows(oldPath, newPath string) bool {
	if moves.movedTo[oldPath] == newPath {
		return true
	}
	// An unmoved file or a missing directory lists nothing; an empty set must not pass vacuously.
	children, err := moves.head.dir(oldPath)
	if err != nil || len(children) == 0 || !moves.arrivesAlone(oldPath, newPath) {
		return false
	}
	return !slices.ContainsFunc(children, func(child string) bool {
		return !moves.pathFollows(child, path.Join(newPath, path.Base(child)))
	})
}

func (moves *moveContext) arrivesAlone(oldPath, newPath string) bool {
	if existing, err := moves.head.dir(newPath); err == nil && len(existing) > 0 {
		return false
	}
	return !slices.ContainsFunc(moves.renames, func(rename pass.Rename) bool {
		rel, isUnder := strings.CutPrefix(rename.NewPath, newPath+"/")
		return isUnder && rename.OldPath != path.Join(oldPath, rel)
	})
}
