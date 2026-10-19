package language

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/sfc"
)

func goStrings() lexer.StringSyntax {
	return lexer.StringSyntax{Delims: []lexer.StringDelim{
		{Open: `"`, Close: `"`, Escapes: true},
		{Open: "`", Close: "`", Multiline: true, Raw: true},
		{Open: `'`, Close: `'`, Escapes: true},
	}}
}

func dartStrings() lexer.StringSyntax {
	interp := "${"
	return lexer.StringSyntax{Delims: []lexer.StringDelim{
		{Open: `"`, Close: `"`, Escapes: true, Interp: interp},
		{Open: `'`, Close: `'`, Escapes: true, Interp: interp},
		{Open: `"""`, Close: `"""`, Escapes: true, Multiline: true, Interp: interp},
		{Open: `'''`, Close: `'''`, Escapes: true, Multiline: true, Interp: interp},
		{Open: `r"`, Close: `"`, Raw: true},
		{Open: `r'`, Close: `'`, Raw: true},
		{Open: `r"""`, Close: `"""`, Multiline: true, Raw: true},
		{Open: `r'''`, Close: `'''`, Multiline: true, Raw: true},
	}}
}

func jsStrings() lexer.StringSyntax {
	interp := "${"
	return lexer.StringSyntax{
		Delims: []lexer.StringDelim{
			{Open: `"`, Close: `"`, Escapes: true, Continues: true},
			{Open: `'`, Close: `'`, Escapes: true, Continues: true},
			{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: interp},
			{
				Open: "`", Close: "`", Escapes: true, Multiline: true,
				Raw: true, Interp: interp, Tag: "String.raw",
			},
			{Open: "/", Close: "/", Escapes: true, Regex: true},
		},
		RegexAfter: lexer.RegexContext{
			After: "(,=:[!&|?{};+-*%<>~^",
			Words: []string{
				"return", "typeof", "case", "in", "of", "void", "yield", "await",
				"delete", "throw", "new",
			},
			ParenWords: []string{"if", "while", "for", "with"},
		},
	}
}

func sfcStrings(regions func(src string) []sfc.Region) lexer.StringSyntax {
	syntax := jsStrings()
	syntax.Within = func(src string) []lexer.Span {
		var scripts []lexer.Span
		for _, region := range regions(src) {
			if region.Tag == "script" {
				scripts = append(scripts, lexer.Span{Start: region.Start, End: region.End})
			}
		}
		return scripts
	}
	return syntax
}
