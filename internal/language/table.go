package language

func languages() []Language {
	return []Language{
		{ID: Go, extensions: []string{".go"}, Comments: cStyle(), packageClause: isGoPackage},
		{ID: Dart, extensions: []string{".dart"}, Comments: dartStyle()},
		{
			ID:         TypeScript,
			extensions: []string{".ts", ".tsx", ".mts", ".cts"},
			Comments:   cStyle(),
		},
		{
			ID:         JavaScript,
			extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
			Comments:   cStyle(),
		},
		{ID: Svelte, extensions: []string{".svelte"}, Comments: componentStyle()},
		{ID: Vue, extensions: []string{".vue"}, Comments: componentStyle()},
		{ID: Java, extensions: []string{".java"}, Comments: cStyle()},
		{ID: Kotlin, extensions: []string{".kt", ".kts"}, Comments: cStyle()},
		{ID: Scala, extensions: []string{".scala"}, Comments: cStyle()},
		{ID: Swift, extensions: []string{".swift"}, Comments: cStyle()},
		{ID: Rust, extensions: []string{".rs"}, Comments: cStyle()},
		{ID: CLang, extensions: []string{".c", ".h"}, Comments: cStyle()},
		{ID: CPP, extensions: []string{".cpp", ".hpp"}, Comments: cStyle()},
		{ID: CSharp, extensions: []string{".cs"}, Comments: cStyle()},
		{ID: Python, extensions: []string{".py"}, Comments: hashStyle()},
		{ID: Ruby, extensions: []string{".rb"}, Comments: hashStyle()},
		{ID: PHP, extensions: []string{".php"}, Comments: cStyle()},
		{ID: Shell, extensions: []string{".sh", ".bash", ".zsh"}, Comments: hashStyle()},
		{ID: SQL, extensions: []string{".sql"}, Comments: sqlStyle(), packageClause: isSQLSchema},
		{ID: Proto, extensions: []string{".proto"}, Comments: cStyle()},
		{ID: Lua, extensions: []string{".lua"}, Comments: luaStyle()},
		{ID: HTML, extensions: []string{".html"}, Comments: markupStyle()},
		{ID: Nix, extensions: []string{".nix"}, Comments: nixStyle()},
		// Config prose is not source: flagging it trains people to ignore the gate.
		{ID: YAML, extensions: []string{".yaml", ".yml"}, Comments: hashStyle(), isConfig: true},
		{ID: TOML, extensions: []string{".toml"}, Comments: hashStyle(), isConfig: true},
	}
}
