package printable

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const ellipsis = "..."

func HeadEnd(text string, limit int, size func(string) int) int {
	used := 0
	pos := 0
	for pos < len(text) {
		_, width := utf8.DecodeRuneInString(text[pos:])
		used += size(text[pos : pos+width])
		if used > limit {
			break
		}
		pos += width
	}
	return pos
}

func lineSize(unit string) int { return len(Line(unit)) }

func Head(text string, limit int) string {
	if lineSize(text) <= limit {
		return text
	}
	return Line(text[:HeadEnd(text, max(limit, 0), lineSize)])
}

func tailStart(text string, limit int, size func(string) int) int {
	used := 0
	pos := len(text)
	for pos > 0 {
		_, width := utf8.DecodeLastRuneInString(text[:pos])
		used += size(text[pos-width : pos])
		if used > max(limit, 0) {
			break
		}
		pos -= width
	}
	return pos
}

func Tail(text string, limit int) string {
	if lineSize(text) <= limit {
		return text
	}
	return Line(text[tailStart(text, limit, lineSize):])
}

type edge int

const headEdge, tailEdge edge = 0, 1

func Shorten(text string, limit int) string {
	return clip(text, limit, len(ellipsis), headEdge)
}

func Fit(text string, limit int) (string, bool) {
	return fit(text, limit, headEdge, escapedWidth)
}

func clip(text string, limit, reserve int, side edge) string {
	kept, whole := fit(text, limit, side, escapedWidth)
	if !whole {
		kept, _ = fit(text, max(limit-reserve, 0), side, escapedWidth)
	}
	switch {
	case whole:
		return Line(kept)
	case side == tailEdge:
		return ellipsis + Line(kept)
	}
	return Line(trimBrokenEscape(kept)) + ellipsis
}

func trimBrokenEscape(kept string) string {
	// A cut backslash or "\x" before the ellipsis would read as a broken escape.
	for {
		cut := strings.LastIndexByte(kept, '\\')
		if cut < 0 || !partialEscape(kept[cut+1:]) {
			return kept
		}
		kept = kept[:cut]
	}
}

func partialEscape(rest string) bool {
	if rest == "" {
		return true
	}
	for _, shape := range []string{"xhh", "uhhhh", "Uhhhhhhhh"} {
		if rest[0] == shape[0] {
			return len(rest) < len(shape) && strings.Trim(rest[1:], "0123456789abcdef") == ""
		}
	}
	return false
}

func fit(text string, limit int, side edge, width func(string) int) (string, bool) {
	lo, hi, used := 0, len(text), 0
	for lo < hi {
		unit := edgeUnit(text[lo:hi], side)
		if used += width(unit); used > limit {
			break
		}
		if side == tailEdge {
			hi -= len(unit)
		} else {
			lo += len(unit)
		}
	}
	switch {
	case lo >= hi:
		return text, true
	case side == tailEdge:
		return text[hi:], false
	}
	return text[:lo], false
}

func edgeUnit(text string, side edge) string {
	if side == tailEdge {
		_, size := utf8.DecodeLastRuneInString(text)
		return text[len(text)-size:]
	}
	_, size := utf8.DecodeRuneInString(text)
	return text[:size]
}

func escapedWidth(unit string) int { return len(Line(unit)) }

func ShortenPath(name string, limit int) string {
	if len(Path(name)) <= limit {
		return name
	}
	// Room for the quotes Path may add and for the ellipsis.
	kept, _ := fit(name, limit-len(`""`)-len(ellipsis), tailEdge, quotedWidth)
	return ellipsis + kept
}

func quotedWidth(unit string) int { return len(strconv.Quote(unit)) - len(`""`) }

const Ellipsis = "..."
