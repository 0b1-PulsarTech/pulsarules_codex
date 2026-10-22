package movepurity

import (
	"fmt"
	"path"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func (moves *moveContext) entries(oldPath, newPath string) (oldEntry, newEntry entry, ok bool) {
	var oldErr, newErr error
	oldEntry, oldErr = moves.head.entryOf(oldPath)
	newEntry, newErr = moves.staged.entryOf(newPath)
	return oldEntry, newEntry, oldErr == nil && newErr == nil
}

type verdict int

const (
	verdictPure verdict = iota
	verdictEdited
	verdictUnsafe
	verdictClauseOnly
	verdictInclusion
	verdictUnlinksLoad
	verdictStrands
	verdictEmbeds
)

func (moves *moveContext) renameVerdict(rename pass.Rename) verdict {
	oldEntry, newEntry, ok := moves.entries(rename.OldPath, rename.NewPath)
	if !ok || oldEntry.mode != newEntry.mode {
		return verdictEdited
	}
	isSame := oldEntry.id == newEntry.id
	if !gosrc.SameInclusion(rename.OldPath, rename.NewPath) {
		if isSame {
			return verdictInclusion
		}
		return verdictEdited
	}
	if isSame {
		return moves.identicalVerdict(rename)
	}
	return moves.isGoEdit(fileEdit{oldPath: rename.OldPath, newPath: rename.NewPath, isMove: true})
}

func (moves *moveContext) identicalVerdict(rename pass.Rename) verdict {
	isGo := isGoFile(rename.OldPath) && isGoFile(rename.NewPath)
	dirs := renameDirs(rename)
	if !isGo && !gosrc.IsCompiled(rename.OldPath) {
		return moves.assetVerdict(rename)
	}
	switch {
	case dirs.oldDir == dirs.newDir,
		gosrc.IsOutsidePackage(rename.OldPath) && gosrc.IsOutsidePackage(rename.NewPath):
		return verdictPure
	case isGo:
		got := moves.isGoEdit(fileEdit{
			oldPath: rename.OldPath, newPath: rename.NewPath, isMove: true, isSame: true,
		})
		if got == verdictPure || got == verdictStrands || got == verdictEmbeds {
			return got
		}
	}
	return verdictUnsafe
}

func (moves *moveContext) assetVerdict(rename pass.Rename) verdict {
	dir := moves.embedCoverDir(moves.head, rename.OldPath)
	if dir == "" {
		dir = moves.embedCoverDir(moves.staged, rename.NewPath)
	}
	if dir == "" {
		return verdictPure
	}
	moves.embedDirs[rename.NewPath] = dir
	return verdictEmbeds
}

func (moves *moveContext) importChangeVerdict(changedPath string) verdict {
	oldEntry, newEntry, ok := moves.entries(changedPath, changedPath)
	if !ok || oldEntry.mode != newEntry.mode {
		return verdictEdited
	}
	if oldEntry.id == newEntry.id {
		return verdictPure
	}
	return moves.isGoEdit(fileEdit{oldPath: changedPath, newPath: changedPath})
}

func (moves *moveContext) isGoEdit(edit fileEdit) verdict {
	if !edit.isGo() {
		return verdictEdited
	}
	oldFile, oldErr := moves.head.parse(edit.oldPath)
	newFile, newErr := moves.staged.parse(edit.newPath)
	if oldErr != nil || newErr != nil {
		return verdictEdited
	}
	return moves.goEquivalent(oldFile, newFile, edit)
}

func notPureMoveFinding(reporter finding.Reporter, rename pass.Rename) finding.Finding {
	return reporter.At(
		rename.NewPath,
		0,
		fmt.Sprintf(
			"rename of %s -> %s is not a pure move: %d%% similar; commit the move, then the edits",
			rename.OldPath, rename.NewPath, rename.Score,
		),
		"stage and commit the rename alone, then commit the content edits separately",
	)
}

func mixedChangesetFinding(reporter finding.Reporter, renameCount, editCount int) finding.Finding {
	return reporter.NewWithSuggestion(
		fmt.Sprintf(
			"this commit mixes %d rename(s) with %d edit(s); split the move into its own commit",
			renameCount, editCount,
		),
		"unstage the unrelated edits, commit the rename(s) alone, then stage and commit the edits",
	)
}

func unlinksLoadFinding(reporter finding.Reporter, file, target string) finding.Finding {
	return reporter.At(
		file,
		0,
		fmt.Sprintf(
			"edit of %s drops the last import of %s, whose init/load-time work stops running",
			file, target,
		),
		fmt.Sprintf(
			"keep a blank import of %s in that file, or move the load-time work first", target,
		),
	)
}

func strandsCompiledFinding(
	reporter finding.Reporter,
	rename pass.Rename,
	file string,
) finding.Finding {
	return reporter.At(
		rename.NewPath,
		0,
		fmt.Sprintf(
			"move of %s -> %s strands or picks up a compiled non-Go file (%s) in %s; "+
				"move it with the package or remove it first",
			rename.OldPath, rename.NewPath, path.Base(file), path.Dir(file),
		),
		"commit the compiled file move together with the package, or remove it first, "+
			"then commit this move with only its package clause and import fixups",
	)
}

func renameCarriesEditFinding(reporter finding.Reporter, rename pass.Rename) finding.Finding {
	return reporter.At(
		rename.NewPath,
		0,
		fmt.Sprintf(
			"rename of %s -> %s carries an edit beyond the move; commit the move, then the edit",
			rename.OldPath, rename.NewPath,
		),
		"stage and commit the rename alone, then commit the content edit separately",
	)
}
