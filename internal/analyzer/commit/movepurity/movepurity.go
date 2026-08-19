package movepurity

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

const analyzerID = "commit-move-purity"

var _ pass.Analyzer = (*Analyzer)(nil)

type Analyzer struct {
	minSimilarity int
	diffs         diffReader
}

func NewAnalyzer(diffs diffReader) *Analyzer {
	const defaultMinSimilarity = 90
	return &Analyzer{minSimilarity: defaultMinSimilarity, diffs: diffs}
}

func (a *Analyzer) ID() string { return analyzerID }

func (a *Analyzer) Stage() pass.StageID { return pass.StageStatic }

func (a *Analyzer) Analyze(ctx *pass.Context) []pass.Finding {
	if len(ctx.StagedRenames) == 0 {
		return nil
	}
	params := ctx.Params(a.ID())
	minSimilarity := params.Int("min_similarity", a.minSimilarity)
	reporter := pass.NewReporter(analyzerID, params.Severity(pass.SeverityWarning), pass.CatCommit)

	var findings []pass.Finding
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
	reporter pass.Reporter,
	rename pass.Rename,
) (pass.Finding, bool) {
	if a.diffs == nil {
		return pass.Finding{}, false
	}
	const renameDiffProbeScore = 1
	diff, err := a.diffs.StagedRenameDiff(rename.OldPath, rename.NewPath, renameDiffProbeScore)
	if err == nil && isImportOnlyDiff(diff) {
		return pass.Finding{}, false
	}
	return renameCarriesEditFinding(reporter, rename), true
}

func (a *Analyzer) unrelatedStagedEdits(
	changes []pass.FileChange,
	renames []pass.Rename,
) []pass.FileChange {
	inRename := make(map[string]bool, len(renames)*2)
	for _, rename := range renames {
		inRename[rename.OldPath] = true
		inRename[rename.NewPath] = true
	}

	var edits []pass.FileChange
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
