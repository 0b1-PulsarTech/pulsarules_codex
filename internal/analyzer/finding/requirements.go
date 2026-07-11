package finding

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

type Category int

const (
	CatSyntax Category = iota
	CatAST
	CatArch
	CatCommit
	CatProject
)

type StageID int

const (
	StageContext StageID = iota
	StageStatic
	StageAST
	StageArch
	StageRuleInjection
	StageOutput
)

type Requirements struct {
	NeedsAST        bool
	NeedsGitHistory bool
}
