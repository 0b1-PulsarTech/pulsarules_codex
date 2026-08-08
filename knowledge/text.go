package knowledge

import (
	"strings"
	"unicode"
)

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

func firstBlockquote(body string) string {
	var quote []string
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
		// The summary is the blockquote before any define; one inside a section is content.
		if strings.HasPrefix(trimmed, "{{define") {
			break
		}
		rest, ok := strings.CutPrefix(trimmed, ">")
		if !ok {
			if quote == nil {
				continue
			}
			break
		}
		quote = append(quote, strings.TrimSpace(rest))
	}
	return strings.Join(quote, " ")
}
