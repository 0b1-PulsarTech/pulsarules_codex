package movepurity

import (
	"path"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity/gosrc"
)

type fileEdit struct {
	oldPath string
	newPath string
	isMove  bool
	isSame  bool
}

func (moves *moveContext) goEquivalent(oldSrc, newSrc string, edit fileEdit) verdict {
	oldFile, oldErr := gosrc.Parse(oldSrc)
	newFile, newErr := gosrc.Parse(newSrc)
	if oldErr != nil || newErr != nil || !moves.sameShape(oldFile, newFile, edit) ||
		!moves.sameBodies(oldFile, newFile, edit) {
		return verdictEdited
	}
	isRelocation := edit.isMove && path.Dir(edit.oldPath) != path.Dir(edit.newPath)
	if isRelocation {
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
	oldDir, newDir := path.Dir(edit.oldPath), path.Dir(edit.newPath)
	scope := gosrc.ImportScope{
		OldMod:   moves.headModule(oldDir),
		NewMod:   moves.stagedModule(newDir),
		DirMoves: moves.dirMoves,
	}
	rules := moves.qualRules(oldFile, newFile, &scope, edit)
	bind := moves.newBinding(oldFile, newFile, scope)
	if edit.isMove && oldDir != newDir {
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
		dirs := [2]string{path.Dir(edit.oldPath), path.Dir(edit.newPath)}
		return moves.renameRules(oldFile, newFile, scope, dirs)
	}
	return moves.dependantRules(oldFile, newFile, scope, path.Dir(edit.oldPath))
}
