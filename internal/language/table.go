package language

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func cBlockComment() []lexer.BlockDelim {
	return []lexer.BlockDelim{{Open: "/*", Close: "*/"}}
}

func specs() []languageSpec {
	return []languageSpec{
		goSpec(), clikeSpec(), hashSpec(), sqlSpec(),
		nixSpec(), luaSpec(), markupSpec(), sfcSpec(),
	}
}

func goSpec() languageSpec {
	return languageSpec{
		Name:               "go",
		Extensions:         []string{".go"},
		CommentPrefixValue: "//",
		Syntax:             lexer.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
		PackageDeclaration: isGoPackageDeclaration,
	}
}

func clikeSpec() languageSpec {
	return languageSpec{
		// Unions Dart/TS/JS/Java/Kt/Scala/Swift/Rust/C/C++/C#/PHP/Proto; excludes
		// .svelte/.vue, whose embedded markup needs "sfc".
		Name: "clike",
		Extensions: []string{
			".dart",
			".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
			".java", ".kt", ".kts", ".scala", ".swift",
			".rs", ".c", ".h", ".cpp", ".hpp", ".cs", ".php", ".proto",
		},
		CommentPrefixValue: "//",
		Syntax:             lexer.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
	}
}

func hashSpec() languageSpec {
	return languageSpec{
		// Covers Python/Ruby/shell/YAML/TOML. Python's triple-quoted """ is a string
		// literal, not a comment, so there is no block delimiter.
		Name: "hash",
		Extensions: []string{
			".py", ".rb", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".toml",
		},
		CommentPrefixValue: "#",
		Syntax:             lexer.CommentSyntax{Line: []string{"#"}},
	}
}

func sqlSpec() languageSpec {
	return languageSpec{
		Name:               "sql",
		Extensions:         []string{".sql"},
		CommentPrefixValue: "--",
		Syntax:             lexer.CommentSyntax{Line: []string{"--"}, Blocks: cBlockComment()},
		PackageDeclaration: isSQLPackageDeclaration,
	}
}

func nixSpec() languageSpec {
	return languageSpec{
		Name:               "nix",
		Extensions:         []string{".nix"},
		CommentPrefixValue: "#",
		Syntax:             lexer.CommentSyntax{Line: []string{"#"}, Blocks: cBlockComment()},
	}
}

func luaSpec() languageSpec {
	return languageSpec{
		Name:               "lua",
		Extensions:         []string{".lua"},
		CommentPrefixValue: "--",
		Syntax: lexer.CommentSyntax{
			Line:   []string{"--"},
			Blocks: []lexer.BlockDelim{{Open: "--[[", Close: "]]"}},
		},
	}
}

func markupSpec() languageSpec {
	return languageSpec{
		// HTML only: core.DefaultSourceExtensions excludes .xml (config/data, not source),
		// so a .xml registration here would be unreachable.
		Name:               "markup",
		Extensions:         []string{".html"},
		CommentPrefixValue: "<!--",
		Syntax: lexer.CommentSyntax{
			Blocks: []lexer.BlockDelim{{Open: "<!--", Close: "-->"}},
		},
	}
}

func sfcSpec() languageSpec {
	return languageSpec{
		// Svelte/Vue unions clike's script grammar with markup's block: one file mixes
		// a "//"/"/* */" script with "<!-- -->" markup and CSS.
		Name:               "sfc",
		Extensions:         []string{".svelte", ".vue"},
		CommentPrefixValue: "//",
		Syntax: lexer.CommentSyntax{
			Line: []string{"//"},
			Blocks: []lexer.BlockDelim{
				{Open: "/*", Close: "*/"},
				{Open: "<!--", Close: "-->"},
			},
		},
	}
}

//nolint:ireturn // table lookup returns the shared interface by design
func Lookup(name string) pass.Language {
	for _, spec := range specs() {
		if spec.Name == name {
			return handler{spec: spec}
		}
	}
	return nil
}

func RegisterAll(reg *pass.LanguageRegistry) {
	for _, spec := range specs() {
		reg.Register(handler{spec: spec})
	}
}
