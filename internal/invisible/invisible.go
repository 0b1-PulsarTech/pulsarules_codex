package invisible

import (
	"unicode"
	"unicode/utf8"
)

type Mark struct {
	Rune   rune
	Class  Class
	Name   string
	Offset int
	Line   int
}

func Scan(src string) []Mark {
	var marks []Mark
	var lookup *tables
	line := 1
	for offset, char := range src {
		if char == '\n' {
			line++
			continue
		}
		if char < utf8.RuneSelf {
			continue
		}
		if lookup == nil {
			lookup = newTables()
		}
		if class, name, found := lookup.classify(char, src, offset); found {
			marks = append(
				marks,
				Mark{Rune: char, Class: class, Name: name, Offset: offset, Line: line},
			)
		}
	}
	return marks
}

func (lookup *tables) classify(char rune, src string, offset int) (Class, string, bool) {
	// Named sets come first so a codepoint's own entry wins over range rules.
	if name, ok := lookup.strip[char]; ok {
		return ClassStrip, name, true
	}
	if name, ok := lookup.space[char]; ok {
		return ClassSpace, name, true
	}
	if name, ok := lookup.contextual[char]; ok {
		return contextualClass(char, src, offset), name, true
	}
	if name, ok := lookup.typographic[char]; ok {
		return ClassTypographic, name, true
	}
	return classifyRange(char)
}

func classifyRange(char rune) (Class, string, bool) {
	const (
		runeLanguageTag          = 0xE0001
		tagRangeFirst            = 0xE0020
		tagRangeLast             = 0xE007F
		privateUseFirst          = 0xE000
		privateUseLast           = 0xF8FF
		privateUsePlaneAFirst    = 0xF0000
		privateUsePlaneALast     = 0xFFFFD
		privateUsePlaneBFirst    = 0x100000
		privateUsePlaneBLast     = 0x10FFFD
		varSelectorFirst         = 0xFE00
		varSelectorLast          = 0xFE0F
		variationSupplementFirst = 0xE0100
		variationSupplementLast  = 0xE01EF
	)
	type runeSpan struct {
		first, last rune
		name        string
	}
	// Tag, private use and variation selectors are too large to enumerate; the
	// closing Cf test catches any format character nobody named.
	spans := []runeSpan{
		{runeLanguageTag, runeLanguageTag, "tag character"},
		{tagRangeFirst, tagRangeLast, "tag character"},
		{privateUseFirst, privateUseLast, "private use character"},
		{privateUsePlaneAFirst, privateUsePlaneALast, "private use character"},
		{privateUsePlaneBFirst, privateUsePlaneBLast, "private use character"},
		{varSelectorFirst, varSelectorLast, "variation selector"},
		{variationSupplementFirst, variationSupplementLast, "variation selector"},
	}
	for _, span := range spans {
		if char >= span.first && char <= span.last {
			return ClassContextual, span.name, true
		}
	}
	if unicode.Is(unicode.Cf, char) {
		return ClassStrip, "unnamed format character", true
	}
	return ClassStrip, "", false
}

func contextualClass(char rune, src string, offset int) Class {
	const (
		runeZWNJ = 0x200C
		runeZWJ  = 0x200D
	)
	if char != runeZWNJ && char != runeZWJ {
		return ClassContextual
	}
	// A joiner flanked by plain ASCII cannot glue an emoji or shape a script.
	if asciiBefore(src, offset) && asciiAfter(src, offset, char) {
		return ClassStrip
	}
	return ClassContextual
}

func asciiBefore(src string, offset int) bool {
	// A marker at offset 0 has no left neighbour, so it stays contextual.
	if offset == 0 {
		return false
	}
	last, _ := utf8.DecodeLastRuneInString(src[:offset])
	return last <= unicode.MaxASCII
}

func asciiAfter(src string, offset int, char rune) bool {
	next, _ := utf8.DecodeRuneInString(src[offset+len(string(char)):])
	return next <= unicode.MaxASCII
}
