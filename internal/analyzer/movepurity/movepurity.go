package movepurity

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

// AnalyzerID is the stable ID this analyzer reports findings under, and the
// key its runtime params are looked up by - exported so the config
// projection in internal/analysis.withMovePurityParams spells it once.
const AnalyzerID = "commit-move-purity"

// defaultMinSimilarity is the git rename-similarity score, out of 100, a
// staged rename must meet to count as a pure move.
const defaultMinSimilarity = 90

// ParamMinSimilarity is the params key config.MovePurityConfig.MinSimilarity
// is projected onto (see internal/analysis.withMovePurityParams) - the one
// place this name is spelled, instead of a string literal on both ends of
// the config-to-param hop.
const ParamMinSimilarity = "min_similarity"

// renameDiffProbeScore is the similarity threshold passed to
// StagedRenameDiff: the pair already came from ctx.StagedRenames, so a low
// probe just needs git to pair it again for the diff rather than re-judge
// whether it counts as a rename at all.
const renameDiffProbeScore = 1

var _ core.Analyzer = (*Analyzer)(nil)

// baseReporter carries this analyzer's id and category; Analyze resolves
// its severity against the run's config each call (see core.Reporter.
// Resolved), the same mechanism every other analyzer's reporter uses.
var baseReporter = core.NewReporter(AnalyzerID, core.SeverityWarning, core.CatCommit)

// Analyzer reports staged renames that are not pure moves: a rename scored
// below the configured similarity, or renames staged alongside unrelated
// edits. Default severity is warning (raise via the "severity" param)
// because a high-similarity rename picking up an import fixup is common;
// the point is to nudge toward committing the move first, not abort it.
type Analyzer struct {
	minSimilarity int
	diffs         diffReader
}

// NewAnalyzer creates a move-purity analyzer with the default similarity
// threshold. diffs may be nil (e.g. no repository is available), in which
// case the analyzer falls back to judging purity from git's own
// rename-similarity score alone: it cannot inspect a file's own diff, so
// every staged change outside a rename still counts as an edit.
func NewAnalyzer(diffs diffReader) *Analyzer {
	return &Analyzer{minSimilarity: defaultMinSimilarity, diffs: diffs}
}

func (a *Analyzer) ID() string { return AnalyzerID }

func (a *Analyzer) Stage() core.StageID { return core.StageStatic }

// Analyze inspects the staged renames in ctx and reports the ones that are
// not pure moves, plus one finding when renames coexist with unrelated
// staged edits.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if len(ctx.StagedRenames) == 0 {
		return nil
	}
	params := ctx.Params(a.ID())
	minSimilarity := params.Int(ParamMinSimilarity, a.minSimilarity)
	reporter := baseReporter.Resolved(ctx)

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

// why: a failed diff fetch is treated as an edit rather than silently
// skipped, erring toward reporting over a silent false negative; no
// diffReader means the check is simply unavailable.
func (a *Analyzer) renameEditFinding(
	reporter core.Reporter,
	rename core.Rename,
) (core.Finding, bool) {
	if a.diffs == nil {
		return core.Finding{}, false
	}
	diff, err := a.diffs.StagedRenameDiff(rename.OldPath, rename.NewPath, renameDiffProbeScore)
	if err == nil && isImportOnlyDiff(diff) {
		return core.Finding{}, false
	}
	return renameCarriesEditFinding(reporter, rename), true
}

// unrelatedStagedEdits returns staged changes that are neither side of any
// staged rename, whose diff also isn't limited to the move's mechanical
// consequences (import path, package clause, alias) - those ride along
// under the move-first rule. Git status keys a rename by its new path, so
// excluding both sides of every pair isolates what a mixed commit smuggled in.
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

// why: false (count it as an edit) whenever it cannot check, so an
// unreadable diff is never mistaken for a pure fixup.
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
