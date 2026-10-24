package movepurity

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/dartsrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type (
	moveContext struct {
		dirMoves    map[string]string
		moved       map[string]bool
		relocated   map[string]bool
		movedTo     map[string]string
		renames     []pass.Rename
		head        *snapshot
		staged      *snapshot
		declared    map[pkgKey]map[string]bool
		left        map[scopeKey]map[string]bool
		taken       map[scopeKey]map[string]bool
		owned       map[declScope]map[string]bool
		dirNames    map[string]dirPackages
		unlinked    map[string]string
		stranded    map[string]string
		embedDirs   map[string]string
		embedUnread map[string]string
		infos       []gosrc.MoveInfo
		dartRoots   map[*snapshot]dartsrc.Roots
	}
	pkgKey struct {
		dir string
		pkg string
	}
	scopeKey struct {
		scope declScope
		pkg   string
	}
)

func newMoveContext(reader gitReader, renames []pass.Rename) *moveContext {
	moves := &moveContext{
		dirMoves:    make(map[string]string, len(renames)),
		moved:       make(map[string]bool, len(renames)*2),
		relocated:   map[string]bool{},
		movedTo:     make(map[string]string, len(renames)),
		renames:     renames,
		head:        newSnapshot(reader, vcs.Head),
		staged:      newSnapshot(reader, vcs.Staged),
		declared:    map[pkgKey]map[string]bool{},
		left:        map[scopeKey]map[string]bool{},
		taken:       map[scopeKey]map[string]bool{},
		owned:       map[declScope]map[string]bool{},
		dirNames:    map[string]dirPackages{},
		unlinked:    map[string]string{},
		stranded:    map[string]string{},
		embedDirs:   map[string]string{},
		embedUnread: map[string]string{},
		dartRoots:   map[*snapshot]dartsrc.Roots{},
	}
	for _, rename := range renames {
		dirs := renameDirs(rename)
		moves.dirMoves[dirs.oldDir] = dirs.newDir
		moves.moved[rename.OldPath] = true
		moves.moved[rename.NewPath] = true
		moves.movedTo[rename.OldPath] = rename.NewPath
		if dirs.oldDir != dirs.newDir {
			moves.relocated[rename.OldPath] = true
			moves.relocated[rename.NewPath] = true
		}
	}
	return moves
}

func (moves *moveContext) declaredIn(dir, pkg string) map[string]bool {
	return cached(moves.declared, pkgKey{dir: dir, pkg: pkg}, func() map[string]bool {
		set := map[string]bool{}
		head, headErr := moves.head.dir(dir)
		staged, stagedErr := moves.staged.dir(dir)
		if headErr != nil || stagedErr != nil {
			return set
		}
		for _, file := range head {
			if isProductionGo(file) && !moves.moved[file] && slices.Contains(staged, file) {
				moves.head.addDecls(file, pkg, set)
			}
		}
		return set
	})
}

func (moves *moveContext) moveInfos() []gosrc.MoveInfo {
	if moves.infos != nil {
		return moves.infos
	}
	byDirs := map[dirPair]*gosrc.MoveInfo{}
	moves.infos = []gosrc.MoveInfo{}
	for _, rename := range moves.renames {
		if !isGoFile(rename.OldPath) || !isGoFile(rename.NewPath) || isTestFile(rename.OldPath) {
			continue
		}
		_, oldErr := moves.head.blob(rename.OldPath)
		_, newErr := moves.staged.blob(rename.NewPath)
		if oldErr != nil || newErr != nil {
			continue
		}
		dirs := renameDirs(rename)
		moved, ok := byDirs[dirs]
		if !ok {
			oldPkg, _ := moves.head.clause(rename.OldPath)
			newPkg, _ := moves.staged.clause(rename.NewPath)
			moved = &gosrc.MoveInfo{
				OldDir: dirs.oldDir, NewDir: dirs.newDir, Decls: map[string]bool{},
				OldFull: moves.head.module(dirs.oldDir).ImportPath(dirs.oldDir),
				NewFull: moves.staged.module(dirs.newDir).ImportPath(dirs.newDir),
				OldPkg:  oldPkg, NewPkg: newPkg,
			}
			byDirs[dirs] = moved
		}
		moves.head.addDecls(rename.OldPath, moved.OldPkg, moved.Decls)
	}
	for _, moved := range byDirs {
		moves.infos = append(moves.infos, *moved)
	}
	return moves.infos
}

func (moves *moveContext) rootsOf(snap *snapshot) dartsrc.Roots {
	return cached(moves.dartRoots, snap, func() dartsrc.Roots {
		files, err := snap.tree(".")
		if err != nil {
			return dartsrc.Roots{}
		}
		return dartsrc.BuildRoots(files, snap.blob)
	})
}
