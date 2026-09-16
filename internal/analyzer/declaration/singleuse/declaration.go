package singleuse

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

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
	Declarations(ctx *pass.Context) []Declaration
	SoleUser(ctx *pass.Context, decl Declaration) (owner string, ok bool)
}
