package language

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type literalSpec struct {
	Comments  lexer.CommentSyntax
	Strings   lexer.StringSyntax
	Astral    string
	keepAtEOF bool
}

func (spec literalSpec) scan(src string) Literals {
	found := lexer.Literals(src, spec.Comments, spec.Strings)
	// With keepAtEOF a literal that ran to EOF, or an unclosed "${", already hides the rest
	// of the file, and the spans before it are as sure as any.
	if found.Complete || (spec.keepAtEOF && found.ToEOF) {
		return Literals{
			spans:    found.Spans,
			comments: found.Comments,
			regions:  found.Regions,
			astral:   spec.Astral,
		}
	}
	// A literal or comment that never ended hides where every later literal ends.
	return Literals{
		spans:      trimSpans(found.Spans, found.Cut),
		comments:   trimSpans(found.Comments, found.Cut),
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

func ScanComments(ext, src string) []lexer.Span {
	lang := ForExtension(ext)
	if lang.regions != nil {
		return componentComments(lang, src)
	}
	return lineComments(lang.Comments, lang.prose, src)
}

func lineComments(
	syntax lexer.CommentSyntax, quoting lexer.StringSyntax, src string,
) []lexer.Span {
	syntax.LineOpens = lineOpens
	return lexer.Literals(src, syntax, quoting).Comments
}

func componentComments(lang Language, src string) []lexer.Span {
	var found []lexer.Span
	from := 0
	for _, region := range lang.regions(src) {
		markup := lineComments(markupStyle(), lexer.StringSyntax{}, src[from:region.Start])
		found = append(found, shifted(markup, from)...)
		body := src[region.Start:region.End]
		code := lineComments(regionComments(region.Tag), lang.prose, body)
		found = append(found, shifted(code, region.Start)...)
		from = region.End
	}
	markup := lineComments(markupStyle(), lexer.StringSyntax{}, src[from:])
	return append(found, shifted(markup, from)...)
}

func regionComments(tag string) lexer.CommentSyntax {
	syntax := lexer.SlashSyntax()
	if tag == "style" {
		syntax.Line = nil
	}
	return syntax
}

func shifted(spans []lexer.Span, by int) []lexer.Span {
	for at := range spans {
		spans[at].Start += by
		spans[at].End += by
	}
	return spans
}
