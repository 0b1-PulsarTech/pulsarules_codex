package movepurity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const AnalyzerID = "commit-move-purity"

const ParamMinSimilarity = "min_similarity"

var _ core.Analyzer = (*Analyzer)(nil)

type Analyzer struct {
	minSimilarity int
	reader        gitReader
	reporter      core.Reporter
}

func NewAnalyzer(reader gitReader) *Analyzer {
	const defaultMinSimilarity = 90
	return &Analyzer{
		minSimilarity: defaultMinSimilarity,
		reader:        reader,
		reporter:      core.NewReporter(AnalyzerID, core.SeverityWarning, core.CatCommit),
	}
}

func (a *Analyzer) ID() string { return AnalyzerID }

func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if len(ctx.StagedRenames) == 0 {
		return nil
	}
	params := ctx.Params(a.ID())
	minSimilarity := params.Int(ParamMinSimilarity, a.minSimilarity)
	reporter := a.reporter.Resolved(ctx)

	var moves *moveContext
	if a.reader != nil {
		moves = newMoveContext(a.reader, ctx.StagedRenames)
	}
	var findings []core.Finding
	for _, rename := range ctx.StagedRenames {
		if finding, ok := renameFinding(moves, reporter, rename, minSimilarity); ok {
			findings = append(findings, finding)
		}
	}
	if edits := unrelatedStagedEdits(moves, ctx.ChangedFiles, ctx.StagedRenames); len(edits) > 0 {
		findings = append(
			findings,
			mixedChangesetFinding(reporter, len(ctx.StagedRenames), len(edits)),
		)
	}
	return findings
}

func renameFinding(
	moves *moveContext,
	reporter core.Reporter,
	rename core.Rename,
	minSimilarity int,
) (core.Finding, bool) {
	isLow := rename.Score < minSimilarity
	if moves == nil {
		return notPureMoveFinding(reporter, rename), isLow
	}
	switch got := moves.renameVerdict(rename); {
	case got == verdictPure:
		return core.Finding{}, false
	case isLow:
		return notPureMoveFinding(reporter, rename), true
	default:
		return renameCarriesEditFinding(reporter, rename), true
	}
}

func unrelatedStagedEdits(
	moves *moveContext,
	changes []core.FileChange,
	renames []core.Rename,
) []core.FileChange {
	// Git status keys a rename by its new path, so both sides of a pair are skipped.
	inRename := make(map[string]bool, len(renames)*2)
	for _, rename := range renames {
		inRename[rename.OldPath] = true
		inRename[rename.NewPath] = true
	}

	var edits []core.FileChange
	for _, change := range changes {
		changed := change.Path
		if !change.Staged || inRename[changed] {
			continue
		}
		got := verdictEdited
		if moves != nil {
			got = moves.importChangeVerdict(changed)
		}
		if got != verdictPure {
			edits = append(edits, change)
		}
	}
	return edits
}
