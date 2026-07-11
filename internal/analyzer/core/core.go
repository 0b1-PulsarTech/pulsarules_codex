package core

// Analyzer is a single self-contained rule. The pipeline discovers analyzers
// at boot, groups them by Stage, and calls Analyze for each event. An analyzer
// must never block; it returns its findings synchronously.
type Analyzer interface {
	// ID returns a stable unique identifier (e.g. "commit-format").
	ID() string
	// Name returns a short human-readable label.
	Name() string
	// Description returns a one-line explanation of what the analyzer checks.
	Description() string
	// Stage returns the pipeline stage this analyzer belongs to.
	Stage() StageID
	// Category returns the analysis technique category.
	Category() Category
	// Needs declares what the analyzer requires from the pipeline context.
	Needs() Requirements
	// Analyze inspects the context and returns findings. It must not block.
	Analyze(ctx *AnalysisContext) []Finding
}
