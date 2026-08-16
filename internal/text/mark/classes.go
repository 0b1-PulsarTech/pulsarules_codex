package mark

// Class groups a marker codepoint by what a caller may safely do with it.
type Class uint8

const (
	// ClassStrip is removable without reading a single neighbour: the codepoint
	// carries no meaning in Go source or Markdown, whatever surrounds it.
	ClassStrip Class = iota
	// ClassSpace is a space homoglyph, replaced by U+0020 rather than deleted.
	ClassSpace
	// ClassContextual is invisible but may be load-bearing (emoji glue, a script
	// joiner), so it is reported and never rewritten.
	ClassContextual
	// ClassTypographic is visible punctuation an AI model reaches for by default:
	// dashes, ellipsis, curly quotes. Reported, never rewritten - inside a string
	// literal or a fenced block it can be deliberate data.
	ClassTypographic
)

// why: bidi controls are in ClassStrip rather than ClassContextual because a
// source file that reorders its own rendering is the Trojan Source attack
// (CVE-2021-42574), not typography worth preserving.
var stripSet = map[rune]string{
	0x00AD: "soft hyphen",
	0x034F: "combining grapheme joiner",
	0x061C: "Arabic letter mark",
	0x115F: "Hangul choseong filler",
	0x1160: "Hangul jungseong filler",
	0x17B4: "Khmer vowel inherent AQ",
	0x17B5: "Khmer vowel inherent AA",
	0x180E: "Mongolian vowel separator",
	0x200B: "zero width space",
	0x200E: "left-to-right mark",
	0x200F: "right-to-left mark",
	0x202A: "left-to-right embedding",
	0x202B: "right-to-left embedding",
	0x202C: "pop directional formatting",
	0x202D: "left-to-right override",
	0x202E: "right-to-left override",
	0x2060: "word joiner",
	0x2061: "function application",
	0x2062: "invisible times",
	0x2063: "invisible separator",
	0x2064: "invisible plus",
	0x2066: "left-to-right isolate",
	0x2067: "right-to-left isolate",
	0x2068: "first strong isolate",
	0x2069: "pop directional isolate",
	0x206A: "inhibit symmetric swapping",
	0x206B: "activate symmetric swapping",
	0x206C: "inhibit Arabic form shaping",
	0x206D: "activate Arabic form shaping",
	0x206E: "national digit shapes",
	0x206F: "nominal digit shapes",
	0xFFF9: "interlinear annotation anchor",
	0xFFFA: "interlinear annotation separator",
	0xFFFB: "interlinear annotation terminator",
}

var spaceSet = map[rune]string{
	0x00A0: "no-break space",
	0x1680: "ogham space mark",
	0x2000: "en quad",
	0x2001: "em quad",
	0x2002: "en space",
	0x2003: "em space",
	0x2004: "three-per-em space",
	0x2005: "four-per-em space",
	0x2006: "six-per-em space",
	0x2007: "figure space",
	0x2008: "punctuation space",
	0x2009: "thin space",
	0x200A: "hair space",
	0x202F: "narrow no-break space",
	0x205F: "medium mathematical space",
	0x3000: "ideographic space",
}

// why: ZWJ/ZWNJ and the variation selectors glue emoji and shape Persian and
// Devanagari, so deciding they are carriers needs the neighbours. This repo has
// no such content today, but a rewrite that guesses wrong corrupts text silently
// - which is worse than a report a human reads.
var contextualSet = map[rune]string{
	0x200C: "zero width non-joiner",
	0x200D: "zero width joiner",
	0xFEFF: "byte order mark",
	0x180B: "Mongolian free variation selector-1",
	0x180C: "Mongolian free variation selector-2",
	0x180D: "Mongolian free variation selector-3",
}

var typographicSet = map[rune]string{
	0x2013: "en dash",
	0x2014: "em dash",
	0x2015: "horizontal bar",
	0x2018: "left single quotation mark",
	0x2019: "right single quotation mark",
	0x201C: "left double quotation mark",
	0x201D: "right double quotation mark",
	0x2026: "horizontal ellipsis",
}
