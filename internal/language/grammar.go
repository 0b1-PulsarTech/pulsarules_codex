package language

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"

func cStyle() lexer.CommentSyntax {
	return lexer.CommentSyntax{
		Line:   []string{"//"},
		Blocks: []lexer.BlockDelim{{Open: "/*", Close: "*/"}},
	}
}

func dartStyle() lexer.CommentSyntax {
	// Dart block comments nest, unlike the C family's.
	syntax := cStyle()
	syntax.Blocks = []lexer.BlockDelim{{Open: "/*", Close: "*/", Nested: true}}
	return syntax
}

func hashStyle() lexer.CommentSyntax {
	return lexer.CommentSyntax{Line: []string{"#"}}
}

func sqlStyle() lexer.CommentSyntax {
	syntax := cStyle()
	syntax.Line = []string{"--"}
	return syntax
}

func nixStyle() lexer.CommentSyntax {
	syntax := cStyle()
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
	syntax := cStyle()
	syntax.Blocks = append(syntax.Blocks, lexer.BlockDelim{Open: "<!--", Close: "-->"})
	return syntax
}
