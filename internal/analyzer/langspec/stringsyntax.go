package langspec

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type Literals struct {
	spans      []lexical.Span
	comments   []lexical.Span
	regions    []lexical.Span
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
	ext = strings.ToLower(ext)
	for _, spec := range literalSpecs() {
		if slices.Contains(spec.Extensions, ext) {
			for _, option := range options {
				option(&spec)
			}
			return spec.scan(src)
		}
	}
	return Literals{}
}

func (found Literals) Covers(offset int) bool { return lexical.InSpans(found.regions, offset) }

func (found Literals) Incomplete() bool { return found.incomplete }

func (found Literals) UnreadFrom() int { return found.unreadFrom }

func (found Literals) Contains(offset int) bool { return lexical.InSpans(found.spans, offset) }

func (found Literals) InComment(offset int) bool { return lexical.InSpans(found.comments, offset) }

func (found Literals) Raw(offset int) bool {
	span, inside := lexical.SpanAt(found.spans, offset)
	return inside && span.Raw
}

func (found Literals) Regex(offset int) bool {
	span, inside := lexical.SpanAt(found.spans, offset)
	return inside && span.Regex
}

func (found Literals) Escape(char rune) string {
	const bmpLimit = 0x10000
	if char < bmpLimit {
		return fmt.Sprintf(`\u%04X`, char)
	}
	return fmt.Sprintf(found.astral, char)
}

func goStrings() lexical.StringSyntax {
	return lexical.StringSyntax{Delims: []lexical.StringDelim{
		{Open: `"`, Close: `"`, Escapes: true},
		{Open: "`", Close: "`", Multiline: true, Raw: true},
		{Open: `'`, Close: `'`, Escapes: true},
	}}
}
