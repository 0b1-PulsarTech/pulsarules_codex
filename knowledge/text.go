package knowledge

import "unicode"

func FirstSentence(text string) string {
	dot := -1
	for offset, char := range text {
		// A dot is one byte, so the next rune visited is the one right after it.
		if dot >= 0 {
			if unicode.IsSpace(char) {
				return text[:dot+1]
			}
			dot = -1
		}
		if char == '.' {
			dot = offset
		}
	}
	if dot >= 0 {
		return text[:dot+1]
	}
	return text
}
