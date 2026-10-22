package movepurity

import (
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
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

func (moves *moveContext) embedsFollow(patterns []string, dirs dirPair) bool {
	if len(patterns) == 0 {
		return true
	}
	// A failed listing of the old directory leaves no files, which never passes below.
	oldFiles, _ := moves.head.tree(dirs.oldDir)
	newFiles, err := moves.head.tree(dirs.newDir)
	if err != nil {
		return false
	}
	before := gosrc.EmbedFiles(patterns, dirs.oldDir, oldFiles)
	after := gosrc.EmbedFiles(patterns, dirs.newDir, moves.stagedFiles(newFiles))
	// Each covered old file must land at its twin and the staged side must cover exactly those
	// names: an equal count would let a twin lost to a nested module hide behind another file.
	return len(before) > 0 && moves.renamedAlong(before, dirs) && maps.Equal(before, after)
}

func (moves *moveContext) renamedAlong(files map[string]bool, dirs dirPair) bool {
	for rel := range files {
		if moves.movedTo[path.Join(dirs.oldDir, rel)] != path.Join(dirs.newDir, rel) {
			return false
		}
	}
	return true
}

func (moves *moveContext) stagedFiles(headFiles []string) []string {
	// A moves-only commit leaves the staged tree equal to HEAD plus its renames.
	files := slices.DeleteFunc(slices.Clone(headFiles), func(file string) bool {
		_, isGone := moves.movedTo[file]
		return isGone
	})
	for _, rename := range moves.renames {
		files = append(files, rename.NewPath)
	}
	return files
}

func (moves *moveContext) embedCoverDir(snap *snapshot, file string) (coverDir, unread string) {
	rest := path.Base(file)
	for dir := path.Dir(file); ; dir = path.Dir(dir) {
		files, err := snap.dir(dir)
		if err != nil {
			return dir, dir + "/"
		}
		if covered, unreadable := moves.embedsIn(snap, files, rest); covered {
			return dir, unreadable
		}
		if dir == "." || slices.Contains(files, path.Join(dir, "go.mod")) {
			return "", ""
		}
		rest = path.Join(path.Base(dir), rest)
	}
}

func (moves *moveContext) embedsIn(
	snap *snapshot,
	files []string,
	rel string,
) (isCovered bool, unread string) {
	for _, file := range files {
		if !isGoFile(file) || moves.relocated[file] || gosrc.IsOutsidePackage(file) {
			continue
		}
		// An unreadable or unparsable embedder counts: reporting beats a silent false negative.
		src, err := snap.blob(file)
		if err != nil {
			return true, file
		}
		if !strings.Contains(src, "go:embed") {
			continue
		}
		parsed, err := snap.parse(file)
		if err != nil {
			return true, file
		}
		if !parsed.IsIgnored && gosrc.EmbedCovers(parsed.Embeds, rel) {
			return true, ""
		}
	}
	return false, ""
}
