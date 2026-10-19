package language

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type Literals struct {
	spans      []lexer.Span
	comments   []lexer.Span
	regions    []lexer.Span
	astral     string
	incomplete bool
	unreadFrom int
}

type ScanOption func(*literalSpec)

func StringsEndAtNewline() ScanOption {
	return func(spec *literalSpec) { spec.Strings.EndAtNewline = true }
}

func KeepSpansAtEOF() ScanOption {
	return func(spec *literalSpec) { spec.keepAtEOF = true }
}

func ScanLiterals(ext, src string, options ...ScanOption) Literals {
	lang := ForExtension(ext)
	if len(lang.strings.Delims) == 0 {
		return Literals{}
	}
	spec := literalSpec{Comments: lang.Comments, Strings: lang.strings, Astral: lang.astral}
	for _, option := range options {
		option(&spec)
	}
	return spec.scan(src)
}

func (found Literals) Covers(offset int) bool { return lexer.InSpans(found.regions, offset) }

func (found Literals) Incomplete() bool { return found.incomplete }

func (found Literals) UnreadFrom() int { return found.unreadFrom }

func (found Literals) Contains(offset int) bool { return lexer.InSpans(found.spans, offset) }

func (found Literals) InComment(offset int) bool { return lexer.InSpans(found.comments, offset) }

func (found Literals) Raw(offset int) bool {
	span, inside := lexer.SpanAt(found.spans, offset)
	return inside && span.Raw
}

func (found Literals) Regex(offset int) bool {
	span, inside := lexer.SpanAt(found.spans, offset)
	return inside && span.Regex
}

func (found Literals) Escape(char rune) string {
	const bmpLimit = 0x10000
	if char < bmpLimit {
		return fmt.Sprintf(`\u%04X`, char)
	}
	return fmt.Sprintf(found.astral, char)
}
