package language

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type literalSpec struct {
	Comments   lexer.CommentSyntax
	Extensions []string
	Strings    lexer.StringSyntax
	Astral     string
}

func (spec literalSpec) scan(src string) Literals {
	found := lexer.Literals(src, spec.Comments, spec.Strings)
	if found.Complete {
		return Literals{spans: found.Spans, regions: found.Regions, astral: spec.Astral}
	}
	// A literal or comment that never ended hides where every later literal ends.
	return Literals{
		spans:      trimSpans(found.Spans, found.Cut),
		regions:    trimRegions(found.Regions, found.Cut),
		astral:     spec.Astral,
		incomplete: true,
		unreadFrom: unreadLine(src, found),
	}
}

func unreadLine(src string, found lexer.Result) int {
	if found.ToEOF {
		return 0
	}
	return 1 + strings.Count(src[:found.Cut], "\n")
}

func literalSpecs() []literalSpec {
	const braced = `\u{%X}`
	return []literalSpec{
		{
			Comments:   goSpec().Syntax,
			Extensions: []string{".go"},
			Strings:    goStrings(),
			Astral:     `\U%08X`,
		},
		{
			Comments:   dartComments(),
			Extensions: []string{".dart"},
			Strings:    dartStrings(),
			Astral:     braced,
		},
		{
			Comments: clikeSpec().Syntax,
			Extensions: []string{
				".js", ".mjs", ".cjs", ".jsx", ".ts", ".mts", ".cts", ".tsx",
			},
			Strings: jsStrings(),
			Astral:  braced,
		},
		{
			Comments:   sfcSpec().Syntax,
			Extensions: []string{".svelte", ".vue"},
			Strings:    sfcStrings(),
			Astral:     braced,
		},
	}
}

func dartComments() lexer.CommentSyntax {
	// Dart block comments nest, unlike the C family's.
	syntax := clikeSpec().Syntax
	syntax.Blocks = []lexer.BlockDelim{{Open: "/*", Close: "*/", Nested: true}}
	return syntax
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

func sfcStrings() lexer.StringSyntax {
	syntax := jsStrings()
	syntax.Within = lexer.BlockDelim{Open: "<script", Close: "</script>"}
	return syntax
}

func trimSpans(spans []lexer.Span, cut int) []lexer.Span {
	var kept []lexer.Span
	for _, span := range spans {
		if span.End <= cut {
			kept = append(kept, span)
		}
	}
	return kept
}

func trimRegions(regions []lexer.Span, cut int) []lexer.Span {
	var kept []lexer.Span
	for _, region := range regions {
		if region.Start < cut {
			region.End = min(region.End, cut)
			kept = append(kept, region)
		}
	}
	return kept
}
