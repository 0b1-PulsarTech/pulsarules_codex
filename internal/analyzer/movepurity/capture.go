package movepurity

import (
	"path"
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity/gosrc"
)

type declScope struct {
	oldDir string
	newDir string
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
	return declScope{
		oldDir: path.Dir(edit.oldPath), newDir: path.Dir(edit.newPath),
		isTest: strings.HasSuffix(edit.newPath, "_test.go"),
	}
}

func isProductionGo(file string) bool {
	return strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go")
}

func (scope declScope) includes(file string) bool {
	return isProductionGo(file) || (scope.isTest && strings.HasSuffix(file, ".go"))
}

func (moves *moveContext) leftBehind(scope declScope, pkg string) map[string]bool {
	return moves.cachedDecls("left", scope, pkg, func(set map[string]bool) {
		files, err := moves.reader.HeadDir(scope.oldDir)
		if err != nil {
			return
		}
		for _, file := range files {
			// A file co-moved into newDir takes its names along; one moved elsewhere
			// still bound them at HEAD.
			dest, isMoved := moves.movedTo[file]
			if !scope.includes(file) || (isMoved && path.Dir(dest) == scope.newDir) {
				continue
			}
			if src, readErr := moves.reader.HeadBlob(file); readErr == nil {
				gosrc.CollectDecls(src, pkg, set)
			}
		}
	})
}

func (moves *moveContext) takenInNewDir(scope declScope, pkg string) map[string]bool {
	return moves.cachedDecls("taken", scope, pkg, func(set map[string]bool) {
		files, err := moves.reader.StagedDir(scope.newDir)
		if err != nil {
			return
		}
		for _, file := range files {
			if !scope.includes(file) || moves.isComovedFrom(file, scope.oldDir) {
				continue
			}
			if src, readErr := moves.reader.StagedBlob(file); readErr == nil {
				gosrc.CollectDecls(src, pkg, set)
			}
		}
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

func (moves *moveContext) cachedDecls(
	kind string, scope declScope, pkg string, fill func(set map[string]bool),
) map[string]bool {
	key := strings.Join(
		[]string{kind, scope.oldDir, scope.newDir, pkg, strconv.FormatBool(scope.isTest)}, "\x00",
	)
	if set, ok := moves.decls[key]; ok {
		return set
	}
	set := map[string]bool{}
	moves.decls[key] = set
	fill(set)
	return set
}
