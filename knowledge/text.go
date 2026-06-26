package knowledge

import "strings"

func FirstSentence(text string) string {
	if i := strings.IndexByte(text, '.'); i >= 0 {
		return text[:i+1]
	}
	return text
}
