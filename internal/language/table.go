package language

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/sfc"
)

func languages() []Language {
	rows := append(scriptLanguages(), []Language{
		{ID: Java, extensions: []string{".java"}, Comments: lexer.SlashSyntax()},
		{ID: Kotlin, extensions: []string{".kt", ".kts"}, Comments: lexer.SlashSyntax()},
		{ID: Scala, extensions: []string{".scala"}, Comments: lexer.SlashSyntax()},
		{ID: Swift, extensions: []string{".swift"}, Comments: lexer.SlashSyntax()},
		{ID: Rust, extensions: []string{".rs"}, Comments: lexer.SlashSyntax()},
		{ID: CLang, extensions: []string{".c", ".h"}, Comments: lexer.SlashSyntax()},
		{ID: CPP, extensions: []string{".cpp", ".hpp"}, Comments: lexer.SlashSyntax()},
		{ID: CSharp, extensions: []string{".cs"}, Comments: lexer.SlashSyntax()},
		{ID: Python, extensions: []string{".py"}, Comments: hashStyle()},
		{ID: Ruby, extensions: []string{".rb"}, Comments: hashStyle()},
		{ID: PHP, extensions: []string{".php"}, Comments: lexer.SlashSyntax()},
		{ID: Shell, extensions: []string{".sh", ".bash", ".zsh"}, Comments: hashStyle()},
		{ID: SQL, extensions: []string{".sql"}, Comments: sqlStyle(), packageClause: isSQLSchema},
		{ID: Proto, extensions: []string{".proto"}, Comments: lexer.SlashSyntax()},
		{ID: Lua, extensions: []string{".lua"}, Comments: luaStyle()},
		{ID: HTML, extensions: []string{".html"}, Comments: markupStyle()},
		{ID: Nix, extensions: []string{".nix"}, Comments: nixStyle()},
		// Config prose is not source: flagging it trains people to ignore the gate.
		{ID: YAML, extensions: []string{".yaml", ".yml"}, Comments: hashStyle(), isConfig: true},
		{ID: TOML, extensions: []string{".toml"}, Comments: hashStyle(), isConfig: true},
	}...)
	for at := range rows {
		rows[at].lineStrings = lineStringsFor(rows[at].ID)
	}
	return rows
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
			astral:        goRune,
			packageClause: isGoPackage,
		},
		{
			ID:         Dart,
			extensions: []string{".dart"},
			Comments:   dartStyle(),
			strings:    dartStrings(),
			astral:     braced,
		},
		{
			ID:         TypeScript,
			extensions: []string{".ts", ".tsx", ".mts", ".cts"},
			Comments:   lexer.SlashSyntax(),
			strings:    jsStrings(),
			astral:     braced,
		},
		{
			ID:         JavaScript,
			extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
			Comments:   lexer.SlashSyntax(),
			strings:    jsStrings(),
			astral:     braced,
		},
		{
			ID:         Svelte,
			extensions: []string{".svelte"},
			Comments:   componentStyle(),
			strings:    sfcStrings(sfc.Regions),
			astral:     braced,
			regions:    sfc.Regions,
		},
		{
			ID:         Vue,
			extensions: []string{".vue"},
			Comments:   componentStyle(),
			strings:    sfcStrings(sfc.PlainRegions),
			astral:     braced,
			regions:    sfc.PlainRegions,
		},
	}
}
