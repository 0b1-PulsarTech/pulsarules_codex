package langspec

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type literalSpec struct {
	Comments   lexical.CommentSyntax
	Extensions []string
	Strings    lexical.StringSyntax
	Astral     string
}

func (spec literalSpec) scan(src string) Literals {
	found := lexical.Literals(src, spec.Comments, spec.Strings)
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

func unreadLine(src string, found lexical.Result) int {
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

func dartComments() lexical.CommentSyntax {
	// Dart block comments nest, unlike the C family's.
	syntax := clikeSpec().Syntax
	syntax.Blocks = []lexical.BlockDelim{{Open: "/*", Close: "*/", Nested: true}}
	return syntax
}

func dartStrings() lexical.StringSyntax {
	interp := "${"
	return lexical.StringSyntax{Delims: []lexical.StringDelim{
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

func jsStrings() lexical.StringSyntax {
	interp := "${"
	return lexical.StringSyntax{
		Delims: []lexical.StringDelim{
			{Open: `"`, Close: `"`, Escapes: true, Continues: true},
			{Open: `'`, Close: `'`, Escapes: true, Continues: true},
			{Open: "`", Close: "`", Escapes: true, Multiline: true, Interp: interp},
			{
				Open: "`", Close: "`", Escapes: true, Multiline: true,
				Raw: true, Interp: interp, Tag: "String.raw",
			},
			{Open: "/", Close: "/", Escapes: true, Regex: true},
		},
		RegexAfter: lexical.RegexContext{
			After: "(,=:[!&|?{};+-*%<>~^",
			Words: []string{
				"return", "typeof", "case", "in", "of", "void", "yield", "await",
				"delete", "throw", "new",
			},
			ParenWords: []string{"if", "while", "for", "with"},
		},
	}
}

func sfcStrings() lexical.StringSyntax {
	syntax := jsStrings()
	syntax.Within = lexical.BlockDelim{Open: "<script", Close: "</script>"}
	return syntax
}

func trimSpans(spans []lexical.Span, cut int) []lexical.Span {
	var kept []lexical.Span
	for _, span := range spans {
		if span.End <= cut {
			kept = append(kept, span)
		}
	}
	return kept
}

func trimRegions(regions []lexical.Span, cut int) []lexical.Span {
	var kept []lexical.Span
	for _, region := range regions {
		if region.Start < cut {
			region.End = min(region.End, cut)
			kept = append(kept, region)
		}
	}
	return kept
}
