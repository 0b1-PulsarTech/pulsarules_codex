package language

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/sfc"
)

func languages() []Language {
	return slices.Concat(
		scriptLanguages(), compiledLanguages(), dynamicLanguages(), configLanguages(),
	)
}

func compiledLanguages() []Language {
	return []Language{
		slashLanguage(Java, likeStrings(), ".java"),
		slashLanguage(Kotlin, likeStrings(), ".kt", ".kts"),
		slashLanguage(Scala, likeStrings(), ".scala"),
		slashLanguage(Swift, likeStrings(), ".swift"),
		{
			ID:         Rust,
			extensions: []string{".rs"},
			Comments:   nestedSlashStyle(),
			prose:      rustStrings(),
		},
		slashLanguage(CLang, likeStrings(), ".c", ".h"),
		slashLanguage(CPP, likeStrings(), ".cpp", ".hpp"),
		slashLanguage(CSharp, csharpStrings(), ".cs"),
	}
}

func dynamicLanguages() []Language {
	return []Language{
		{ID: Python, extensions: []string{".py"}, Comments: hashStyle(), prose: scriptStrings()},
		{ID: Ruby, extensions: []string{".rb"}, Comments: hashStyle(), prose: scriptStrings()},
		slashLanguage(PHP, likeStrings(), ".php"),
		{
			ID:         Shell,
			extensions: []string{".sh", ".bash", ".zsh"},
			Comments:   hashStyle(),
			prose:      shellStrings(),
		},
		{
			ID:            SQL,
			extensions:    []string{".sql"},
			Comments:      sqlStyle(),
			packageClause: isSQLSchema,
			prose:         sqlStrings(),
		},
		slashLanguage(Proto, likeStrings(), ".proto"),
		{ID: Lua, extensions: []string{".lua"}, Comments: luaStyle(), prose: scriptStrings()},
		{ID: HTML, extensions: []string{".html"}, Comments: markupStyle()},
		{ID: Nix, extensions: []string{".nix"}, Comments: nixStyle(), prose: scriptStrings()},
	}
}

func configLanguages() []Language {
	return []Language{
		// Config prose is not source: flagging it trains people to ignore the gate.
		{
			ID:         YAML,
			extensions: []string{".yaml", ".yml"},
			Comments:   hashStyle(),
			isConfig:   true,
			prose:      scalarStrings(),
		},
		{
			ID:         TOML,
			extensions: []string{".toml"},
			Comments:   hashStyle(),
			isConfig:   true,
			prose:      scalarStrings(),
		},
	}
}

func scriptLanguages() []Language {
	const (
		braced = `\u{%X}`
		goRune = `\U%08X`
	)
	return []Language{
		{
			ID:            Go,
			extensions:    []string{".go"},
			Comments:      lexer.SlashSyntax(),
			strings:       goStrings(),
			prose:         likeStrings(),
			astral:        goRune,
			packageClause: isGoPackage,
		},
		{
			ID:         Dart,
			extensions: []string{".dart"},
			Comments:   nestedSlashStyle(),
			strings:    dartStrings(),
			prose:      likeStrings(),
			astral:     braced,
		},
		{
			ID:         TypeScript,
			extensions: []string{".ts", ".tsx", ".mts", ".cts"},
			Comments:   lexer.SlashSyntax(),
			strings:    lexer.JSStrings(),
			prose:      likeStrings(),
			astral:     braced,
		},
		{
			ID:         JavaScript,
			extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
			Comments:   lexer.SlashSyntax(),
			strings:    lexer.JSStrings(),
			prose:      likeStrings(),
			astral:     braced,
		},
		{
			ID:         Svelte,
			extensions: []string{".svelte"},
			Comments:   componentStyle(),
			strings:    sfcStrings(sfc.Regions),
			prose:      likeStrings(),
			astral:     braced,
			regions:    sfc.Regions,
		},
		{
			ID:         Vue,
			extensions: []string{".vue"},
			Comments:   componentStyle(),
			strings:    sfcStrings(sfc.PlainRegions),
			prose:      likeStrings(),
			astral:     braced,
			regions:    sfc.PlainRegions,
		},
	}
}

func slashLanguage(id ID, prose lexer.StringSyntax, extensions ...string) Language {
	return Language{ID: id, extensions: extensions, Comments: lexer.SlashSyntax(), prose: prose}
}
