package movepurity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const AnalyzerID = "commit-move-purity"

const ParamMinSimilarity = "min_similarity"

var _ core.Analyzer = (*Analyzer)(nil)

type Analyzer struct {
	minSimilarity int
	diffs         diffReader
	reporter      core.Reporter
}

func NewAnalyzer(diffs diffReader) *Analyzer {
	const defaultMinSimilarity = 90
	return &Analyzer{
		minSimilarity: defaultMinSimilarity,
		diffs:         diffs,
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

	var findings []core.Finding
	for _, rename := range ctx.StagedRenames {
		if rename.Score < minSimilarity {
			findings = append(findings, notPureMoveFinding(reporter, rename))
			continue
		}
		if finding, ok := a.renameEditFinding(reporter, rename); ok {
			findings = append(findings, finding)
		}
	}
	if edits := a.unrelatedStagedEdits(ctx.ChangedFiles, ctx.StagedRenames); len(edits) > 0 {
		findings = append(
			findings,
			mixedChangesetFinding(reporter, len(ctx.StagedRenames), len(edits)),
		)
	}
	return findings
}

func (a *Analyzer) renameEditFinding(
	reporter core.Reporter,
	rename core.Rename,
) (core.Finding, bool) {
	if a.diffs == nil {
		return core.Finding{}, false
	}
	const renameDiffProbeScore = 1
	diff, err := a.diffs.StagedRenameDiff(rename.OldPath, rename.NewPath, renameDiffProbeScore)
	if err == nil && isImportOnlyDiff(diff) {
		return core.Finding{}, false
	}
	return renameCarriesEditFinding(reporter, rename), true
}

func (a *Analyzer) unrelatedStagedEdits(
	changes []core.FileChange,
	renames []core.Rename,
) []core.FileChange {
	inRename := make(map[string]bool, len(renames)*2)
	for _, rename := range renames {
		inRename[rename.OldPath] = true
		inRename[rename.NewPath] = true
	}

	var edits []core.FileChange
	for _, change := range changes {
		if !change.Staged || inRename[change.Path] {
			continue
		}
		if a.isPureImportChange(change.Path) {
			continue
		}
		edits = append(edits, change)
	}
	return edits
}

func (a *Analyzer) isPureImportChange(path string) bool {
	if a.diffs == nil {
		return false
	}
	diff, err := a.diffs.StagedDiff(path)
	if err != nil {
		return false
	}
	return isImportOnlyDiff(diff)
}
