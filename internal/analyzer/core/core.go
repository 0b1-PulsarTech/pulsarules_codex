package core

// Analyzer is a single self-contained rule that never blocks.
//
// why: Name, Description and Category once lived here; only tests read
// them, and Category() could silently disagree with Finding.Category.
// Dropping those 3 dead methods (22 implementations) tightened this.
type Analyzer interface {
	// ID returns a stable unique identifier (e.g. "commit-format").
	ID() string
	// Stage returns the pipeline stage this analyzer belongs to.
	Stage() StageID
	// Analyze inspects the context and returns findings. It must not block.
	Analyze(ctx *AnalysisContext) []Finding
}

// InPlaceAnalyzer is the second, distinct Analyze contract: an analyzer
// that transforms the WHOLE ctx.Findings slice in place (reordering,
// deduplicating, annotating) instead of contributing new findings. Its
// Analyze return carries no meaning - the pipeline never appends it - so
// a caller inspects ctx.Findings afterward, not the return value.
type InPlaceAnalyzer interface {
	Analyzer
	// TransformsInPlace is a marker method: it carries no behavior of its
	// own, existing only to make an analyzer's in-place contract visible in
	// the type system instead of only in a doc comment on Analyze's return.
	TransformsInPlace()
}
