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
	if existing, err := moves.head.dir(newPath); err != nil || len(existing) > 0 {
		return false
	}
	return !slices.ContainsFunc(moves.renames, func(rename pass.Rename) bool {
		rel, isUnder := strings.CutPrefix(rename.NewPath, newPath+"/")
		return isUnder && rename.OldPath != path.Join(oldPath, rel)
	})
}

func (moves *moveContext) embedCoverDir(snap *snapshot, file string) string {
	rest := path.Base(file)
	for dir := path.Dir(file); ; dir = path.Dir(dir) {
		files, err := snap.dir(dir)
		if err != nil || moves.embedsIn(snap, files, rest) {
			return dir
		}
		if dir == "." || slices.Contains(files, path.Join(dir, "go.mod")) {
			return ""
		}
		rest = path.Join(path.Base(dir), rest)
	}
}

func (moves *moveContext) embedsIn(snap *snapshot, files []string, rel string) bool {
	for _, file := range files {
		if !isGoFile(file) || moves.relocated[file] || gosrc.IsOutsidePackage(file) {
			continue
		}
		// An unreadable or unparsable embedder counts: reporting beats a silent false negative.
		src, err := snap.blob(file)
		if err != nil {
			return true
		}
		if !strings.Contains(src, "go:embed") {
			continue
		}
		parsed, err := snap.parse(file)
		if err != nil || (!parsed.IsIgnored && gosrc.EmbedCovers(parsed.Embeds, rel)) {
			return true
		}
	}
	return false
}
