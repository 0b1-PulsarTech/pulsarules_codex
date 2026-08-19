package core

// Analyzer is a single self-contained rule. The pipeline discovers analyzers
// at boot, groups them by Stage, and calls Analyze for each event. An analyzer
// must never block; it returns its findings synchronously.
//
// why: Name, Description and Category used to be part of this interface,
// but the pipeline never read them - only _test.go assertions did, and
// Finding.Category (set by Reporter) could silently disagree with an
// analyzer's own Category() anyway. 22 implementations carried 3 dead
// methods each for metadata nothing production consumed; deleting them
// shrinks the interface to exactly what RunStages reads.
type Analyzer interface {
	// ID returns a stable unique identifier (e.g. "commit-format").
	ID() string
	// Stage returns the pipeline stage this analyzer belongs to.
	Stage() StageID
	// Analyze inspects the context and returns findings. It must not block.
	Analyze(ctx *AnalysisContext) []Finding
}

// InPlaceAnalyzer is the second, distinct Analyze contract: an analyzer
// that transforms the WHOLE ctx.Findings slice already accumulated by
// earlier stages (reordering, deduplicating, annotating) rather than
// contributing new findings of its own. Its Analyze return value carries no
// meaning - the pipeline never appends it - so a caller cannot test one by
// its return value alone; it must inspect ctx.Findings afterward instead.
// ruleinjection.Analyzer and output.Analyzer implement this at
// StageRuleInjection/StageOutput; every other analyzer implements the
// plain, additive Analyzer contract.
type InPlaceAnalyzer interface {
	Analyzer
	// TransformsInPlace is a marker method: it carries no behavior of its
	// own, existing only to make an analyzer's in-place contract visible in
	// the type system instead of only in a doc comment on Analyze's return.
	TransformsInPlace()
}
