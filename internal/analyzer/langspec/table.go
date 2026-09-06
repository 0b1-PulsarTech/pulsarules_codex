package langspec

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"

func cBlockComment() []core.BlockDelim {
	return []core.BlockDelim{{Open: "/*", Close: "*/"}}
}

func specs() []Spec {
	return []Spec{goSpec()}
}

func goSpec() Spec {
	return Spec{
		Name:               "go",
		Extensions:         []string{".go"},
		Syntax:             core.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
		PackageDeclaration: isGoPackageDeclaration,
	}
}

//nolint:ireturn // table lookup returns the shared interface by design
func Lookup(name string) core.Language {
	for _, spec := range specs() {
		if spec.Name == name {
			return handler{spec: spec}
		}
	}
	return nil
}

func RegisterAll(reg *core.LanguageRegistry) {
	for _, spec := range specs() {
		reg.Register(handler{spec: spec})
	}
}
