package printable

import "unicode/utf8"

func Clip(text string, limit int) string {
	const ellipsis = "..."
	kept, whole := fit(text, limit)
	if whole {
		return Line(kept)
	}
	return ellipsis + Line(kept)
}

func fit(text string, limit int) (string, bool) {
	hi, used := len(text), 0
	for hi > 0 {
		_, size := utf8.DecodeLastRuneInString(text[:hi])
		if used += escapedWidth(text[hi-size : hi]); used > limit {
			break
		}
		hi -= size
	}
	if hi == 0 {
		return text, true
	}
	return text[hi:], false
}

func escapedWidth(unit string) int { return len(Line(unit)) }
