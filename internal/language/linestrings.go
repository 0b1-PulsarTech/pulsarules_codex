package language

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"

func lineStringsFor(id ID) lexer.StringSyntax {
	switch id {
	case Rust:
		return rustStrings()
	case CSharp:
		return csharpStrings()
	case Shell:
		return shellStrings()
	case SQL:
		return sqlStrings()
	case Python, Ruby, Lua, Nix:
		return scriptStrings()
	case YAML, TOML:
		return scalarStrings()
	case HTML:
		return lexer.StringSyntax{}
	}
	return likeStrings()
}

func quoted() lexer.StringDelim {
	return lexer.StringDelim{Open: `"`, Close: `"`, Escapes: true}
}

func rawPrefixed(prefix string) lexer.StringDelim {
	return lexer.StringDelim{
		Open: prefix + `"`, Close: `"`, Raw: true, Hashes: true, Opens: noIdentBefore,
	}
}

func likeStrings() lexer.StringSyntax {
	return lexer.StringSyntax{EndAtNewline: true, Delims: []lexer.StringDelim{
		rawPrefixed("r"),
		{Open: `r'`, Close: `'`, Raw: true, Whole: true, Opens: noIdentBefore},
		quoted(),
		{Open: `'`, Close: `'`, Escapes: true, Whole: true, Opens: wordOrNone},
		{Open: "`", Close: "`", Escapes: true, Multiline: true},
	}}
}

func rustStrings() lexer.StringSyntax {
	return lexer.StringSyntax{EndAtNewline: true, Delims: []lexer.StringDelim{
		rawPrefixed("r"),
		rawPrefixed("br"),
		rawPrefixed("cr"),
		quoted(),
		{Open: `'`, Close: `'`, Escapes: true, Char: true},
	}}
}

func csharpStrings() lexer.StringSyntax {
	syntax := likeStrings()
	for _, open := range []string{`@"`, `$@"`, `@$"`} {
		syntax.Delims = append(syntax.Delims, lexer.StringDelim{
			Open: open, Close: `"`, Doubled: true,
		})
	}
	return syntax
}

func shellStrings() lexer.StringSyntax {
	return lexer.StringSyntax{EndAtNewline: true, Delims: []lexer.StringDelim{
		quoted(),
		{Open: `'`, Close: `'`, Whole: true},
	}}
}

func sqlStrings() lexer.StringSyntax {
	return lexer.StringSyntax{EndAtNewline: true, Delims: []lexer.StringDelim{
		{Open: `"`, Close: `"`},
		{Open: `'`, Close: `'`, Whole: true},
	}}
}

func scriptStrings() lexer.StringSyntax {
	return lexer.StringSyntax{EndAtNewline: true, Delims: []lexer.StringDelim{
		quoted(),
		{Open: `'`, Close: `'`, Escapes: true, Whole: true},
	}}
}

func scalarStrings() lexer.StringSyntax {
	return lexer.StringSyntax{EndAtNewline: true, Delims: []lexer.StringDelim{
		{Open: `"`, Close: `"`, Escapes: true, Opens: scalarStart},
		{Open: `'`, Close: `'`, Whole: true, Opens: scalarStart},
	}}
}
