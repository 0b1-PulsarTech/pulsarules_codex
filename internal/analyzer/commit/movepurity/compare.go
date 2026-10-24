package movepurity

import (
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/dartsrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type (
	dirPair struct {
		oldDir string
		newDir string
	}
	fileEdit struct {
		oldPath string
		newPath string
		isMove  bool
		isSame  bool
	}
)

func isGoFile(file string) bool { return strings.HasSuffix(file, ".go") }

func isTestFile(file string) bool { return strings.HasSuffix(file, "_test.go") }

func isProductionGo(file string) bool { return isGoFile(file) && !isTestFile(file) }

func renameDirs(rename pass.Rename) dirPair {
	return dirPair{oldDir: path.Dir(rename.OldPath), newDir: path.Dir(rename.NewPath)}
}

func (edit fileEdit) dirs() dirPair {
	return dirPair{oldDir: path.Dir(edit.oldPath), newDir: path.Dir(edit.newPath)}
}

func (edit fileEdit) isGo() bool { return isGoFile(edit.oldPath) && isGoFile(edit.newPath) }

func (edit fileEdit) isRelocation() bool {
	dirs := edit.dirs()
	return edit.isMove && dirs.oldDir != dirs.newDir
}

func (moves *moveContext) goEquivalent(oldFile, newFile gosrc.File, edit fileEdit) verdict {
	if !moves.sameShape(oldFile, newFile, edit) || !moves.sameBodies(oldFile, newFile, edit) {
		return verdictEdited
	}
	if edit.isRelocation() {
		if got := moves.relocationVerdict(oldFile, newFile, edit); got != verdictPure {
			return got
		}
	}
	if target := moves.unlinkedImport(oldFile, newFile); target != "" {
		moves.unlinked[edit.newPath] = target
		return verdictUnlinksLoad
	}
	return verdictPure
}

func (moves *moveContext) sameBodies(oldFile, newFile gosrc.File, edit fileEdit) bool {
	dirs := edit.dirs()
	scope := gosrc.ImportScope{
		OldMod:   moves.head.module(dirs.oldDir),
		NewMod:   moves.staged.module(dirs.newDir),
		DirMoves: moves.dirMoves,
	}
	rules := moves.qualRules(oldFile, newFile, &scope, edit)
	bind := moves.newBinding(oldFile, newFile, scope)
	if edit.isRelocation() {
		bind.Captures = moves.captureSets(oldFile, newFile, edit)
	}
	return scope.Allowed(oldFile.Imports, newFile.Imports) &&
		gosrc.BodiesEquivalent(oldFile, newFile, rules, bind)
}

func (moves *moveContext) sameShape(oldFile, newFile gosrc.File, edit fileEdit) bool {
	if !edit.isMove && oldFile.Pkg != newFile.Pkg {
		return false
	}
	if !gosrc.PreambleMatches(oldFile, newFile) || !slices.Equal(oldFile.Header, newFile.Header) {
		return false
	}
	// An R100 move can leave several package clauses in one directory mid-series.
	return !edit.isMove || edit.isSame || moves.packageAllowed(oldFile, newFile, edit)
}

func (moves *moveContext) qualRules(
	oldFile, newFile gosrc.File, scope *gosrc.ImportScope, edit fileEdit,
) []gosrc.QualRule {
	if edit.isMove {
		return moves.renameRules(oldFile, newFile, scope, edit.dirs())
	}
	return moves.dependantRules(oldFile, newFile, scope, edit.dirs().oldDir)
}

func isDartFile(file string) bool { return strings.HasSuffix(file, ".dart") }

func (moves *moveContext) dartEdit(edit fileEdit) verdict {
	if !isDartFile(edit.oldPath) || !isDartFile(edit.newPath) {
		return verdictEdited
	}
	oldSrc, oldErr := moves.head.blob(edit.oldPath)
	newSrc, newErr := moves.staged.blob(edit.newPath)
	if oldErr != nil || newErr != nil {
		return verdictEdited
	}
	oldFile, oldParseErr := dartsrc.Parse(oldSrc)
	newFile, newParseErr := dartsrc.Parse(newSrc)
	if oldParseErr != nil || newParseErr != nil {
		return verdictEdited
	}
	pair := dartsrc.Pair{
		Old: dartsrc.Source{File: oldFile, Path: edit.oldPath, Roots: moves.rootsFunc(moves.head)},
		New: dartsrc.Source{
			File:  newFile,
			Path:  edit.newPath,
			Roots: moves.rootsFunc(moves.staged),
		},
	}
	// A re-sort is no consequence of a move: a file that differs needs a rewritten URI.
	isEquivalent, rewrites := dartsrc.Equivalent(pair, moves.followsMove)
	if isEquivalent && (rewrites > 0 || edit.isSame) {
		return verdictPure
	}
	return verdictEdited
}

func (moves *moveContext) rootsFunc(snap *snapshot) func() dartsrc.Roots {
	return func() dartsrc.Roots { return moves.rootsOf(snap) }
}

func (moves *moveContext) identicalMove(rename pass.Rename) verdict {
	if isDartFile(rename.OldPath) && isDartFile(rename.NewPath) {
		return moves.dartIdentical(rename)
	}
	return moves.identicalVerdict(rename)
}

func (moves *moveContext) dartIdentical(rename pass.Rename) verdict {
	if got := moves.coveredVerdict(rename); got != verdictPure {
		return got
	}
	edit := fileEdit{oldPath: rename.OldPath, newPath: rename.NewPath, isMove: true, isSame: true}
	if moves.dartEdit(edit) == verdictPure {
		return verdictPure
	}
	return verdictUnsafe
}

func (moves *moveContext) isGoEdit(edit fileEdit) verdict {
	if !edit.isGo() {
		return moves.dartEdit(edit)
	}
	oldFile, oldErr := moves.head.parse(edit.oldPath)
	newFile, newErr := moves.staged.parse(edit.newPath)
	if oldErr != nil || newErr != nil {
		return verdictEdited
	}
	return moves.goEquivalent(oldFile, newFile, edit)
}

func (moves *moveContext) followsMove(oldTarget, newTarget string) bool {
	if moved, ok := moves.movedTo[oldTarget]; ok {
		return newTarget == moved
	}
	return newTarget == oldTarget
}
