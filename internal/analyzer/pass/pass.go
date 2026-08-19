package pass

type Analyzer interface {
	ID() string
	Stage() StageID
	Needs() Requirements
	Analyze(ctx *Context) []Finding
}
