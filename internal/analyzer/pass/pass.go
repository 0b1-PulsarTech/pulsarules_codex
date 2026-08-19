package pass

type Analyzer interface {
	ID() string
	Stage() StageID
	Analyze(ctx *Context) []Finding
}
