package language

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"

func cBlockComment() []pass.BlockDelim {
	return []pass.BlockDelim{{Open: "/*", Close: "*/"}}
}

func specs() []Spec {
	return []Spec{
		goSpec(), clikeSpec(), hashSpec(), sqlSpec(),
		luaSpec(), markupSpec(), sfcSpec(),
	}
}

func goSpec() Spec {
	return Spec{
		Name:               "go",
		Extensions:         []string{".go"},
		Syntax:             pass.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
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
		Syntax: pass.CommentSyntax{Line: []string{"//"}, Blocks: cBlockComment()},
	}
}

func hashSpec() Spec {
	return Spec{
		Name: "hash",
		Extensions: []string{
			".py", ".rb", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".toml",
		},
		Syntax: pass.CommentSyntax{Line: []string{"#"}},
	}
}

func sqlSpec() Spec {
	return Spec{
		Name:               "sql",
		Extensions:         []string{".sql"},
		Syntax:             pass.CommentSyntax{Line: []string{"--"}, Blocks: cBlockComment()},
		PackageDeclaration: isSQLPackageDeclaration,
	}
}

func luaSpec() Spec {
	return Spec{
		Name:       "lua",
		Extensions: []string{".lua"},
		Syntax: pass.CommentSyntax{
			Line:   []string{"--"},
			Blocks: []pass.BlockDelim{{Open: "--[[", Close: "]]"}},
		},
	}
}

func markupSpec() Spec {
	return Spec{
		Name:       "markup",
		Extensions: []string{".html"},
		Syntax: pass.CommentSyntax{
			Blocks: []pass.BlockDelim{{Open: "<!--", Close: "-->"}},
		},
	}
}

func sfcSpec() Spec {
	return Spec{
		Name:       "sfc",
		Extensions: []string{".svelte", ".vue"},
		Syntax: pass.CommentSyntax{
			Line: []string{"//"},
			Blocks: []pass.BlockDelim{
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
