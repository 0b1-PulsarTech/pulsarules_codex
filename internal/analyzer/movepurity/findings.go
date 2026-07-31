package movepurity

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

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
