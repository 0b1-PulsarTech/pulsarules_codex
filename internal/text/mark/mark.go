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
func classifyRange(r rune) (Class, string, bool) {
	switch {
	case r == 0xE0001 || (r >= 0xE0020 && r <= 0xE007F):
		return ClassContextual, "tag character", true
	case r >= 0xE000 && r <= 0xF8FF,
		r >= 0xF0000 && r <= 0xFFFFD,
		r >= 0x100000 && r <= 0x10FFFD:
		return ClassContextual, "private use character", true
	case r >= 0xFE00 && r <= 0xFE0F, r >= 0xE0100 && r <= 0xE01EF:
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
	if r != 0x200C && r != 0x200D {
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
