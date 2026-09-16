package core

type DeclKind string

const (
	DeclConst DeclKind = "const"
	DeclVar   DeclKind = "var"
)

type InitKind string

const (
	InitNone      InitKind = "none"
	InitCall      InitKind = "call"
	InitComposite InitKind = "composite"
	InitLiteral   InitKind = "literal"
)

type Declaration struct {
	Name        string
	Kind        DeclKind
	Exported    bool
	TypeName    string
	Annotations []string
	Init        InitKind
	File        string
	Line        int
}

type DeclarationProvider interface {
	Declarations(ctx *AnalysisContext) []Declaration
	SoleUser(ctx *AnalysisContext, decl Declaration) (owner string, ok bool)
}
