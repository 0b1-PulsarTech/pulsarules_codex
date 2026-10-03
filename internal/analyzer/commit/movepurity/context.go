package movepurity

import (
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type moveContext struct {
	reader   gitReader
	dirMoves map[string]string
	moved    map[string]bool
	movedTo  map[string]string
	decls    map[string]map[string]bool
	dirNames map[string]dirPackages
	pkgNames map[moduleKey]string
	modules  map[moduleKey]gosrc.Module
	unlinked map[string]string
	stranded map[string]string
	renames  []pass.Rename
	infos    []gosrc.MoveInfo
}

func newMoveContext(reader gitReader, renames []pass.Rename) *moveContext {
	ctx := &moveContext{
		reader:   newCachedReader(reader),
		dirMoves: make(map[string]string, len(renames)),
		moved:    make(map[string]bool, len(renames)*2),
		movedTo:  make(map[string]string, len(renames)),
		decls:    map[string]map[string]bool{},
		dirNames: map[string]dirPackages{},
		pkgNames: map[moduleKey]string{},
		modules:  map[moduleKey]gosrc.Module{},
		unlinked: map[string]string{},
		stranded: map[string]string{},
		renames:  renames,
	}
	for _, rename := range renames {
		ctx.dirMoves[path.Dir(rename.OldPath)] = path.Dir(rename.NewPath)
		ctx.moved[rename.OldPath] = true
		ctx.moved[rename.NewPath] = true
		ctx.movedTo[rename.OldPath] = rename.NewPath
	}
	return ctx
}

func (moves *moveContext) declared(dir, pkg string) map[string]bool {
	key := dir + "\x00" + pkg
	if set, ok := moves.decls[key]; ok {
		return set
	}
	set := map[string]bool{}
	moves.decls[key] = set
	head, headErr := moves.reader.HeadDir(dir)
	staged, stagedErr := moves.reader.StagedDir(dir)
	if headErr != nil || stagedErr != nil {
		return set
	}
	for _, file := range head {
		isGo := strings.HasSuffix(file, ".go") && !strings.HasSuffix(file, "_test.go")
		if !isGo || moves.moved[file] || !slices.Contains(staged, file) {
			continue
		}
		if src, err := moves.reader.HeadBlob(file); err == nil {
			gosrc.CollectDecls(src, pkg, set)
		}
	}
	return set
}

func (moves *moveContext) moveInfos() []gosrc.MoveInfo {
	if moves.infos != nil {
		return moves.infos
	}
	byDirs := map[[2]string]*gosrc.MoveInfo{}
	moves.infos = []gosrc.MoveInfo{}
	for _, rename := range moves.renames {
		if path.Ext(rename.OldPath) != ".go" || path.Ext(rename.NewPath) != ".go" ||
			strings.HasSuffix(rename.OldPath, "_test.go") {
			continue
		}
		oldSrc, oldErr := moves.reader.HeadBlob(rename.OldPath)
		newSrc, newErr := moves.reader.StagedBlob(rename.NewPath)
		if oldErr != nil || newErr != nil {
			continue
		}
		dirs := [2]string{path.Dir(rename.OldPath), path.Dir(rename.NewPath)}
		moved, ok := byDirs[dirs]
		if !ok {
			moved = &gosrc.MoveInfo{
				OldDir: dirs[0], NewDir: dirs[1], Decls: map[string]bool{},
				OldFull: moves.headModule(dirs[0]).ImportPath(dirs[0]),
				NewFull: moves.stagedModule(dirs[1]).ImportPath(dirs[1]),
				OldPkg:  packageOf(oldSrc), NewPkg: packageOf(newSrc),
			}
			byDirs[dirs] = moved
		}
		gosrc.CollectDecls(oldSrc, moved.OldPkg, moved.Decls)
	}
	for _, moved := range byDirs {
		moves.infos = append(moves.infos, *moved)
	}
	return moves.infos
}

func packageOf(src string) string {
	name, _ := gosrc.PackageClause(src)
	return name
}

type moduleKey struct {
	isStaged bool
	dir      string
}

func (moves *moveContext) headModule(dir string) gosrc.Module {
	return moves.cachedModule(moduleKey{dir: dir}, moves.reader.HeadBlob)
}

func (moves *moveContext) stagedModule(dir string) gosrc.Module {
	return moves.cachedModule(moduleKey{isStaged: true, dir: dir}, moves.reader.StagedBlob)
}

func (moves *moveContext) cachedModule(
	key moduleKey,
	read func(string) (string, error),
) gosrc.Module {
	if mod, ok := moves.modules[key]; ok {
		return mod
	}
	mod := gosrc.WalkModule(key.dir, read)
	moves.modules[key] = mod
	return mod
}
