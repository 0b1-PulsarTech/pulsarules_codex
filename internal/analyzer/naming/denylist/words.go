package denylist

import (
	"path"
	"slices"
	"strings"
	"unicode"
)

func splitWords(text string) []string {
	runes := []rune(text)
	var words []string
	start := -1
	flush := func(end int) {
		if start >= 0 {
			words = append(words, strings.ToLower(string(runes[start:end])))
			start = -1
		}
	}
	for pos, char := range runes {
		if !isWordRune(char) {
			flush(pos)
			continue
		}
		if start >= 0 && startsWord(runes, pos) {
			flush(pos)
		}
		if start < 0 {
			start = pos
		}
	}
	flush(len(runes))
	return words
}

func startsWord(runes []rune, pos int) bool {
	if !unicode.IsUpper(runes[pos]) {
		return false
	}
	back := pos - 1
	for back > 0 && unicode.Is(unicode.Mn, runes[back]) {
		back--
	}
	prev := runes[back]
	if unicode.IsLower(prev) || unicode.IsDigit(prev) {
		return true
	}
	return unicode.IsUpper(prev) && pos+1 < len(runes) && unicode.IsLower(runes[pos+1])
}

func isWordRune(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || unicode.Is(unicode.Mn, char)
}

type matcher struct {
	tokens  [][]string
	allowed []string
}

func newMatcher(tokens, allowed []string) matcher {
	phrases := make([][]string, 0, len(tokens))
	for _, token := range tokens {
		if words := splitWords(token); len(words) > 0 {
			phrases = append(phrases, words)
		}
	}
	return matcher{tokens: phrases, allowed: allowed}
}

func (found matcher) hitFile(base string) (string, bool) {
	if slices.Contains(found.allowed, base) {
		return "", false
	}
	stem := strings.TrimSuffix(base, path.Ext(base))
	if stem == "" {
		stem = base
	}
	return found.hit(stem)
}

func (found matcher) hit(name string) (string, bool) {
	if slices.Contains(found.allowed, name) {
		return "", false
	}
	words := splitWords(name)
	for _, phrase := range found.tokens {
		if containsRun(words, phrase) {
			return strings.Join(phrase, " "), true
		}
	}
	return "", false
}

func containsRun(words, phrase []string) bool {
	for start := 0; start+len(phrase) <= len(words); start++ {
		if slices.Equal(words[start:start+len(phrase)], phrase) {
			return true
		}
	}
	return false
}
