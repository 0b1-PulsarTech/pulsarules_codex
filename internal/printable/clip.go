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

func Shorten(text string, limit int) string {
	if _, whole := Fit(text, limit); whole {
		return Line(text)
	}
	kept := text[:HeadEnd(text, max(limit-len(ellipsis), 0), lineSize)]
	return Line(trimBrokenEscape(kept)) + ellipsis
}

func Fit(text string, limit int) (string, bool) {
	end := HeadEnd(text, limit, lineSize)
	return text[:end], end == len(text)
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

func ShortenPath(name string, limit int) string {
	if len(Path(name)) <= limit {
		return name
	}
	// Room for the quotes Path may add and for the ellipsis.
	return ellipsis + name[tailStart(name, limit-len(`""`)-len(ellipsis), quotedWidth):]
}

func quotedWidth(unit string) int { return len(strconv.Quote(unit)) - len(`""`) }
