package language

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"

func nestedSlashStyle() lexer.CommentSyntax {
	// Dart and Rust block comments nest, unlike the C family's.
	syntax := lexer.SlashSyntax()
	syntax.Blocks[0].Nested = true
	return syntax
}

func hashStyle() lexer.CommentSyntax {
	return lexer.CommentSyntax{Line: []string{"#"}}
}

func sqlStyle() lexer.CommentSyntax {
	syntax := lexer.SlashSyntax()
	syntax.Line = []string{"--"}
	return syntax
}

func nixStyle() lexer.CommentSyntax {
	syntax := lexer.SlashSyntax()
	syntax.Line = []string{"#"}
	return syntax
}

func luaStyle() lexer.CommentSyntax {
	return lexer.CommentSyntax{
		Line:   []string{"--"},
		Blocks: []lexer.BlockDelim{{Open: "--[[", Close: "]]"}},
	}
}

func markupStyle() lexer.CommentSyntax {
	return lexer.CommentSyntax{Blocks: []lexer.BlockDelim{{Open: "<!--", Close: "-->"}}}
}

func componentStyle() lexer.CommentSyntax {
	// One file mixes a "//" and "/* */" script with "<!-- -->" markup and CSS.
	syntax := lexer.SlashSyntax()
	syntax.Blocks = append(syntax.Blocks, lexer.BlockDelim{Open: "<!--", Close: "-->"})
	return syntax
}
