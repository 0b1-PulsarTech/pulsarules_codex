package movepurity

import (
	"fmt"
	"path"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity/gosrc"
)

type entry struct {
	mode string
	id   string
}

func (moves *moveContext) entries(oldPath, newPath string) (oldEntry, newEntry entry, ok bool) {
	oldMode, oldID, oldErr := moves.reader.HeadEntry(oldPath)
	newMode, newID, newErr := moves.reader.StagedEntry(newPath)
	if oldErr != nil || newErr != nil {
		return entry{}, entry{}, false
	}
	return entry{oldMode, oldID}, entry{newMode, newID}, true
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
)

func (moves *moveContext) renameVerdict(rename core.Rename) verdict {
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

func (moves *moveContext) identicalVerdict(rename core.Rename) verdict {
	isGo := path.Ext(rename.OldPath) == ".go" && path.Ext(rename.NewPath) == ".go"
	switch {
	case path.Dir(rename.OldPath) == path.Dir(rename.NewPath),
		gosrc.IsBuildIgnored(rename.OldPath) && gosrc.IsBuildIgnored(rename.NewPath):
		return verdictPure
	case !isGo && !gosrc.IsCompiled(rename.OldPath):
		return verdictPure
	case isGo:
		got := moves.isGoEdit(fileEdit{
			oldPath: rename.OldPath, newPath: rename.NewPath, isMove: true, isSame: true,
		})
		if got == verdictPure || got == verdictStrands {
			return got
		}
	}
	return verdictUnsafe
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
	if path.Ext(edit.oldPath) != ".go" || path.Ext(edit.newPath) != ".go" {
		return verdictEdited
	}
	oldSrc, oldErr := moves.reader.HeadBlob(edit.oldPath)
	newSrc, newErr := moves.reader.StagedBlob(edit.newPath)
	if oldErr != nil || newErr != nil {
		return verdictEdited
	}
	return moves.goEquivalent(oldSrc, newSrc, edit)
}

func notPureMoveFinding(reporter core.Reporter, rename core.Rename) core.Finding {
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

func mixedChangesetFinding(reporter core.Reporter, renameCount, editCount int) core.Finding {
	return reporter.NewWithSuggestion(
		fmt.Sprintf(
			"this commit mixes %d rename(s) with %d edit(s); split the move into its own commit",
			renameCount, editCount,
		),
		"unstage the unrelated edits, commit the rename(s) alone, then stage and commit the edits",
	)
}

func unlinksLoadFinding(reporter core.Reporter, file, target string) core.Finding {
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

func strandsCompiledFinding(reporter core.Reporter, rename core.Rename, file string) core.Finding {
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

func renameCarriesEditFinding(reporter core.Reporter, rename core.Rename) core.Finding {
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
