package invisible

type Class uint8

func (class Class) Removable() bool { return class == ClassStrip || class == ClassSpace }

const (
	ClassStrip Class = iota
	ClassSpace
	ClassContextual
	ClassTypographic
)

func stripNames() map[rune]string {
	return map[rune]string{
		'\u00AD': "soft hyphen",
		'\u034F': "combining grapheme joiner",
		'\u061C': "Arabic letter mark",
		'\u115F': "Hangul choseong filler",
		'\u1160': "Hangul jungseong filler",
		'\u17B4': "Khmer vowel inherent AQ",
		'\u17B5': "Khmer vowel inherent AA",
		'\u180E': "Mongolian vowel separator",
		'\u200B': "zero width space",
		'\u200E': "left-to-right mark",
		'\u200F': "right-to-left mark",
		'\u202A': "left-to-right embedding",
		'\u202B': "right-to-left embedding",
		'\u202C': "pop directional formatting",
		'\u202D': "left-to-right override",
		'\u202E': "right-to-left override",
		'\u2060': "word joiner",
		'\u2061': "function application",
		'\u2062': "invisible times",
		'\u2063': "invisible separator",
		'\u2064': "invisible plus",
		'\u2066': "left-to-right isolate",
		'\u2067': "right-to-left isolate",
		'\u2068': "first strong isolate",
		'\u2069': "pop directional isolate",
		'\u206A': "inhibit symmetric swapping",
		'\u206B': "activate symmetric swapping",
		'\u206C': "inhibit Arabic form shaping",
		'\u206D': "activate Arabic form shaping",
		'\u206E': "national digit shapes",
		'\u206F': "nominal digit shapes",
		'\uFFF9': "interlinear annotation anchor",
		'\uFFFA': "interlinear annotation separator",
		'\uFFFB': "interlinear annotation terminator",
	}
}

func spaceNames() map[rune]string {
	return map[rune]string{
		'\u00A0': "no-break space",
		'\u1680': "ogham space mark",
		'\u2000': "en quad",
		'\u2001': "em quad",
		'\u2002': "en space",
		'\u2003': "em space",
		'\u2004': "three-per-em space",
		'\u2005': "four-per-em space",
		'\u2006': "six-per-em space",
		'\u2007': "figure space",
		'\u2008': "punctuation space",
		'\u2009': "thin space",
		'\u200A': "hair space",
		'\u202F': "narrow no-break space",
		'\u205F': "medium mathematical space",
		'\u3000': "ideographic space",
	}
}

func contextualNames() map[rune]string {
	return map[rune]string{
		'\u200C': "zero width non-joiner",
		'\u200D': "zero width joiner",
		'\uFEFF': "byte order mark",
		'\u180B': "Mongolian free variation selector-1",
		'\u180C': "Mongolian free variation selector-2",
		'\u180D': "Mongolian free variation selector-3",
	}
}

func typographicNames() map[rune]string {
	return map[rune]string{
		'\u2013': "en dash",
		'\u2014': "em dash",
		'\u2015': "horizontal bar",
		'\u2018': "left single quotation mark",
		'\u2019': "right single quotation mark",
		'\u201C': "left double quotation mark",
		'\u201D': "right double quotation mark",
		'\u2026': "horizontal ellipsis",
	}
}

type tables struct {
	strip       map[rune]string
	space       map[rune]string
	contextual  map[rune]string
	typographic map[rune]string
}

func newTables() *tables {
	return &tables{
		// Bidi controls are ClassStrip because reordering rendering is Trojan
		// Source (CVE-2021-42574), not typography worth preserving.
		strip: stripNames(),
		space: spaceNames(),
		// ZWJ/ZWNJ and variation selectors glue emoji and shape scripts, so a wrong
		// guess corrupts text silently.
		contextual:  contextualNames(),
		typographic: typographicNames(),
	}
}
