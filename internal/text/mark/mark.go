package mark

import (
	"unicode"
	"unicode/utf8"
)

// Mark is one marker occurrence, located by byte offset so a caller can report
// a line or splice the rune out without re-scanning.
type Mark struct {
	Rune   rune
	Class  Class
	Name   string
	Offset int
	Line   int
}

// Scan reports every marker in src, in byte order.
func Scan(src string) []Mark {
	var marks []Mark
	line := 1
	for offset, r := range src {
		if r == '\n' {
			line++
			continue
		}
		if class, name, found := classify(r, src, offset); found {
			marks = append(
				marks,
				Mark{Rune: r, Class: class, Name: name, Offset: offset, Line: line},
			)
		}
	}
	return marks
}

// why: the named sets come first so a codepoint's own entry wins over the
// broader range rules that follow.
func classify(r rune, src string, offset int) (Class, string, bool) {
	if name, ok := stripSet[r]; ok {
		return ClassStrip, name, true
	}
	if name, ok := spaceSet[r]; ok {
		return ClassSpace, name, true
	}
	if name, ok := contextualSet[r]; ok {
		return contextualClass(r, src, offset), name, true
	}
	if name, ok := typographicSet[r]; ok {
		return ClassTypographic, name, true
	}
	return classifyRange(r)
}

// why: tag characters, the private use planes and the variation selectors are
// too large to enumerate, and the trailing Cf test is the closing rule - a
// format character nobody named is still a carrier.

// The Unicode boundaries classifyRange and contextualClass turn on, named so a
// reader need not look each one up.
const (
	runeLanguageTag  = 0xE0001
	tagRangeFirst    = 0xE0020
	tagRangeLast     = 0xE007F
	privateUseFirst  = 0xE000
	privateUseLast   = 0xF8FF
	privatePlaneAF   = 0xF0000
	privatePlaneALst = 0xFFFFD
	privatePlaneBF   = 0x100000
	privatePlaneBLst = 0x10FFFD
	varSelectorFirst = 0xFE00
	varSelectorLast  = 0xFE0F
	varSupplementF   = 0xE0100
	varSupplementL   = 0xE01EF
	// why: the joiners are the only carriers whose neighbours decide whether
	// they are load-bearing.
	runeZWNJ = 0x200C
	runeZWJ  = 0x200D
)

func classifyRange(r rune) (Class, string, bool) {
	switch {
	case r == runeLanguageTag || (r >= tagRangeFirst && r <= tagRangeLast):
		return ClassContextual, "tag character", true
	case r >= privateUseFirst && r <= privateUseLast,
		r >= privatePlaneAF && r <= privatePlaneALst,
		r >= privatePlaneBF && r <= privatePlaneBLst:
		return ClassContextual, "private use character", true
	case r >= varSelectorFirst && r <= varSelectorLast,
		r >= varSupplementF && r <= varSupplementL:
		return ClassContextual, "variation selector", true
	case unicode.Is(unicode.Cf, r):
		return ClassStrip, "unnamed format character", true
	}
	return ClassStrip, "", false
}

// why: a joiner whose neighbours are both plain ASCII cannot be gluing an emoji
// or shaping a script, so it is a carrier and safe to strip. Anything else keeps
// its neighbours' benefit of the doubt.
func contextualClass(r rune, src string, offset int) Class {
	if r != runeZWNJ && r != runeZWJ {
		return ClassContextual
	}
	if asciiBefore(src, offset) && asciiAfter(src, offset, r) {
		return ClassStrip
	}
	return ClassContextual
}

// why: a marker at offset 0 has no left neighbour, so it cannot be flanked by
// ASCII and stays contextual - the caller must not strip what it cannot judge.
func asciiBefore(src string, offset int) bool {
	if offset == 0 {
		return false
	}
	last, _ := utf8.DecodeLastRuneInString(src[:offset])
	return last < unicode.MaxASCII
}

func asciiAfter(src string, offset int, r rune) bool {
	rest := src[offset+len(string(r)):]
	for _, next := range rest {
		return next < unicode.MaxASCII
	}
	return false
}
