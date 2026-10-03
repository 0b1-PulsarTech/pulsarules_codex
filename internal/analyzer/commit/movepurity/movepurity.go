package movepurity

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

const ID = "commit-move-purity"

const ParamMinSimilarity = "min_similarity"

var _ pass.Analyzer = (*Analyzer)(nil)

type Analyzer struct {
	minSimilarity int
	reader        gitReader
	reporter      pass.Reporter
}

func NewAnalyzer(reader gitReader) *Analyzer {
	const defaultMinSimilarity = 90
	return &Analyzer{
		minSimilarity: defaultMinSimilarity,
		reader:        reader,
		reporter:      pass.NewReporter(ID, pass.SeverityWarning, pass.CatCommit),
	}
}

func (analyzer *Analyzer) ID() string { return ID }

func (analyzer *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if len(ctx.StagedRenames) == 0 {
		return nil
	}
	params := ctx.Params(analyzer.ID())
	minSimilarity := params.Int(ParamMinSimilarity, analyzer.minSimilarity)
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)

	var moves *moveContext
	if analyzer.reader != nil {
		moves = newMoveContext(analyzer.reader, ctx.StagedRenames)
	}
	var findings []pass.Finding
	for _, rename := range ctx.StagedRenames {
		if finding, ok := renameFinding(moves, reporter, rename, minSimilarity); ok {
			findings = append(findings, finding)
		}
	}
	edits, unlinks := unrelatedStagedEdits(moves, ctx.ChangedFiles, ctx.StagedRenames)
	for _, unlink := range unlinks {
		findings = append(findings, unlinksLoadFinding(reporter, unlink, moves.unlinked[unlink]))
	}
	if len(edits) > 0 {
		findings = append(
			findings,
			mixedChangesetFinding(reporter, len(ctx.StagedRenames), len(edits)),
		)
	}
	return findings
}

func renameFinding(
	moves *moveContext,
	reporter pass.Reporter,
	rename pass.Rename,
	minSimilarity int,
) (pass.Finding, bool) {
	isLow := rename.Score < minSimilarity
	if moves == nil {
		return notPureMoveFinding(reporter, rename), isLow
	}
	switch got := moves.renameVerdict(rename); {
	case got == verdictPure:
		return pass.Finding{}, false
	case got == verdictClauseOnly:
		return relocationUnsafeFinding(
			reporter,
			rename,
			"differs only by mechanical package/import fixups but is",
		), true
	case got == verdictStrands:
		return strandsCompiledFinding(reporter, rename, moves.stranded[rename.NewPath]), true
	case isLow:
		return notPureMoveFinding(reporter, rename), true
	default:
		return editFinding(moves, reporter, rename, got), true
	}
}

func editFinding(
	moves *moveContext, reporter pass.Reporter, rename pass.Rename, got verdict,
) pass.Finding {
	switch got {
	case verdictUnsafe:
		return relocationUnsafeFinding(reporter, rename, "is byte-identical but")
	case verdictInclusion:
		return inclusionChangeFinding(reporter, rename)
	case verdictUnlinksLoad:
		return unlinksLoadFinding(reporter, rename.NewPath, moves.unlinked[rename.NewPath])
	}
	// A failed git read lands here too: reporting beats a silent false negative.
	return renameCarriesEditFinding(reporter, rename)
}

func unrelatedStagedEdits(
	moves *moveContext,
	changes []pass.FileChange,
	renames []pass.Rename,
) (edits []pass.FileChange, unlinks []string) {
	// Git status keys a rename by its new path, so both sides of a pair are skipped.
	inRename := make(map[string]bool, len(renames)*2)
	for _, rename := range renames {
		inRename[rename.OldPath] = true
		inRename[rename.NewPath] = true
	}

	for _, change := range changes {
		changed := change.Path
		if !change.Staged || inRename[changed] {
			continue
		}
		got := verdictEdited
		if moves != nil {
			got = moves.importChangeVerdict(changed)
		}
		switch got {
		case verdictPure:
		case verdictUnlinksLoad:
			unlinks = append(unlinks, changed)
		default:
			edits = append(edits, change)
		}
	}
	return edits, unlinks
}
