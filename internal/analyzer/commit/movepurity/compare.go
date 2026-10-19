package movepurity

import (
	"path"
	"slices"
	"strings"

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
