package knowledge

import (
	"strings"
	"unicode"
)

// FirstSentence returns text up to and including the first sentence-ending
// period, or text unchanged when it has none. A period ends a sentence only
// when followed by whitespace or end of string; one inside a dotted
// identifier (_test.go, http.DefaultClient) is always followed by more
// identifier characters, so it is skipped instead of mistaken for a boundary.
func FirstSentence(text string) string {
	dot := -1
	for i, r := range text {
		// why: dot's byte length is 1, so range's next visited index is always
		// dot+1 - the rune range already decoded right after the candidate dot.
		if dot >= 0 {
			if unicode.IsSpace(r) {
				return text[:dot+1]
			}
			dot = -1
		}
		if r == '.' {
			dot = i
		}
	}
	if dot >= 0 {
		return text[:dot+1]
	}
	return text
}

// why: a rule body carries its distilled summary as a "> " block under the H1,
// before any {{define}} section; a blockquote written inside a section (e.g.
// one authored under {{define "forbidden"}}) is section content, not the
// summary, so scanning stops at the first {{define}} line.
func firstBlockquote(body string) string {
	var quote []string
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimSpace(line)
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
