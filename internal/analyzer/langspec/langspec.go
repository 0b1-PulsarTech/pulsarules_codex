package langspec

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type Spec struct {
	Name               string
	Extensions         []string
	CommentPrefixValue string
	Syntax             lexical.CommentSyntax
	PackageDeclaration func(line string) bool
}

type handler struct{ spec Spec }

var _ core.Language = handler{}

func (language handler) Extensions() []string { return language.spec.Extensions }

func (language handler) CommentPrefix() string { return language.spec.CommentPrefixValue }

func (language handler) CommentSyntax() lexical.CommentSyntax { return language.spec.Syntax }

func (language handler) IsPackageDeclaration(line string) bool {
	// Most languages have no line analogous to Go's package clause.
	if language.spec.PackageDeclaration == nil {
		return false
	}
	return language.spec.PackageDeclaration(line)
}
