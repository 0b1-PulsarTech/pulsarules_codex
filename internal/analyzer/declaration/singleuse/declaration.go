package singleuse

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

type declKind string

const (
	declConst declKind = "const"
	declVar   declKind = "var"
)

type initKind string

const (
	initCall      initKind = "call"
	initComposite initKind = "composite"
	initLiteral   initKind = "literal"
)

type declaration struct {
	Name        string
	Kind        declKind
	Exported    bool
	TypeName    string
	Annotations []string
	Init        initKind
	File        string
	Line        int
}

type declarationSource interface {
	Declarations(ctx *pass.Context) []declaration
	SoleUser(ctx *pass.Context, decl declaration) (owner string, ok bool)
}
