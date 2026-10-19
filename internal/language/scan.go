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
