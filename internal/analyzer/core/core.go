package core

type Analyzer interface {
	ID() string
	Stage() StageID
	Analyze(ctx *AnalysisContext) []Finding
}

type InPlaceAnalyzer interface {
	Analyzer
	TransformsInPlace()
}
