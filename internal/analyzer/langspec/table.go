package langspec

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

func cBlockComment() []lexical.BlockDelim {
	return []lexical.BlockDelim{{Open: "/*", Close: "*/"}}
}

func specs() []Spec {
	return []Spec{
		goSpec(), clikeSpec(), hashSpec(), sqlSpec(),
		nixSpec(), luaSpec(), markupSpec(), sfcSpec(),
	}
}

func goSpec() Spec {
	return Spec{
		Name:               "go",
		Extensions:         []string{".go"},
		CommentPrefixValue: "//",
		Syntax:             lexical.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
		PackageDeclaration: isGoPackageDeclaration,
	}
}

func clikeSpec() Spec {
	return Spec{
		Name: "clike",
		Extensions: []string{
			".dart",
			".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
			".java", ".kt", ".kts", ".scala", ".swift",
			".rs", ".c", ".h", ".cpp", ".hpp", ".cs", ".php", ".proto",
		},
		CommentPrefixValue: "//",
		Syntax:             lexical.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
	}
}

func hashSpec() Spec {
	return Spec{
		Name: "hash",
		Extensions: []string{
			".py", ".rb", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".toml",
		},
		CommentPrefixValue: "#",
		Syntax:             lexical.CommentSyntax{Line: []string{"#"}},
	}
}

func sqlSpec() Spec {
	return Spec{
		Name:               "sql",
		Extensions:         []string{".sql"},
		CommentPrefixValue: "--",
		Syntax:             lexical.CommentSyntax{Line: []string{"--"}, Blocks: cBlockComment()},
		PackageDeclaration: isSQLPackageDeclaration,
	}
}

func nixSpec() Spec {
	return Spec{
		Name:               "nix",
		Extensions:         []string{".nix"},
		CommentPrefixValue: "#",
		Syntax:             lexical.CommentSyntax{Line: []string{"#"}, Blocks: cBlockComment()},
	}
}

func luaSpec() Spec {
	return Spec{
		Name:               "lua",
		Extensions:         []string{".lua"},
		CommentPrefixValue: "--",
		Syntax: lexical.CommentSyntax{
			Line:   []string{"--"},
			Blocks: []lexical.BlockDelim{{Open: "--[[", Close: "]]"}},
		},
	}
}

func markupSpec() Spec {
	return Spec{
		Name:               "markup",
		Extensions:         []string{".html"},
		CommentPrefixValue: "<!--",
		Syntax: lexical.CommentSyntax{
			Blocks: []lexical.BlockDelim{{Open: "<!--", Close: "-->"}},
		},
	}
}

func sfcSpec() Spec {
	return Spec{
		Name:               "sfc",
		Extensions:         []string{".svelte", ".vue"},
		CommentPrefixValue: "//",
		Syntax: lexical.CommentSyntax{
			Line: []string{"//"},
			Blocks: []lexical.BlockDelim{
				{Open: "/*", Close: "*/"},
				{Open: "<!--", Close: "-->"},
			},
		},
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
