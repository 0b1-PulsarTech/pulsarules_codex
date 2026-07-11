package core

type Analyzer interface {
	ID() string
	Stage() StageID
	Needs() Requirements
	Analyze(ctx *AnalysisContext) []Finding
}
