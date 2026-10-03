package narration

import (
	"slices"
	"strings"
	"unicode"
)

type segment struct {
	line int
	text string
}

type word struct {
	text string
	line int
	end  bool
}

type hit struct {
	phrase string
	line   int
}

func narrationPhrases() [][]string {
	const phraseCapacity = 32
	phrases := make([]string, 0, phraseCapacity)
	phrases = append(phrases,
		"was replaced by",
		"used to be",
		"used to call",
		"used to return",
		"used to live",
		"used to use",
		"the old code",
		"the old implementation",
		"the old approach",
		"previously we",
		"previously this",
		"previously it",
		"previously had",
		"was changed to",
		"no longer needed",
		"no longer used",
		"no longer calls",
		"no longer returns",
		"never ever used to *",
	)
	for _, subject := range []string{"it", "this", "we", "they", "never"} {
		phrases = append(phrases, subject+" used to *")
	}
	words := make([][]string, 0, len(phrases))
	for _, phrase := range phrases {
		words = append(words, strings.Fields(phrase))
	}
	return words
}

func wordsOf(segments []segment) []word {
	var words []word
	for _, seg := range segments {
		for raw := range strings.FieldsSeq(seg.text) {
			normalized := strings.Map(func(char rune) rune {
				if unicode.IsLetter(char) || unicode.IsDigit(char) {
					return unicode.ToLower(char)
				}
				return ' '
			}, raw)
			start := len(words)
			for field := range strings.FieldsSeq(normalized) {
				words = append(words, word{text: field, line: seg.line})
			}
			trimmed := strings.TrimRight(raw, ")]\"'\u201d\u2019\u00bb\u203a")
			if len(words) > start && strings.ContainsAny(trimmed[len(trimmed)-1:], ".!?") {
				words[len(words)-1].end = true
			}
		}
	}
	return words
}

func hasPhraseAt(words []word, phrase []string) bool {
	needed := len(phrase)
	if phrase[needed-1] == "*" {
		needed--
	}
	if len(words) < needed {
		return false
	}
	for offset, want := range phrase[:needed] {
		if want != "*" && words[offset].text != want || offset < needed-1 && words[offset].end {
			return false
		}
	}
	return true
}

func passiveUsed(words []word, start int, phrase []string) bool {
	if phrase[0] != "used" && phrase[0] != "never" {
		return false
	}
	adverbs := []string{
		"only", "also", "just", "still", "mainly", "mostly", "never", "rarely", "often",
		"solely", "primarily", "not", "t", "ever", "always", "usually", "typically", "currently",
		"generally", "commonly", "sometimes",
	}
	auxiliaries := []string{
		"is", "are", "was", "were", "be", "been", "being",
		"s", "re", "isn", "aren", "wasn", "weren",
	}
	const maxAdverbs = 2
	for back, skipped := start-1, 0; back >= 0; back, skipped = back-1, skipped+1 {
		text := words[back].text
		if words[back].end {
			return false
		}
		if slices.Contains(auxiliaries, text) {
			return true
		}
		if skipped == maxAdverbs || !slices.Contains(adverbs, text) {
			return false
		}
	}
	return false
}

func findHits(words []word, phrases [][]string) []hit {
	var hits []hit
	for start := 0; start < len(words); start++ {
		index := slices.IndexFunc(phrases, func(phrase []string) bool {
			return hasPhraseAt(words[start:], phrase) && !passiveUsed(words, start, phrase)
		})
		if index < 0 {
			continue
		}
		size := min(len(phrases[index]), len(words)-start)
		matched := make([]string, 0, size)
		for _, found := range words[start : start+size] {
			matched = append(matched, found.text)
		}
		hits = append(hits, hit{phrase: strings.Join(matched, " "), line: words[start].line})
		start += size - 1
	}
	return hits
}

func blocks(segments []segment) [][]segment {
	var groups [][]segment
	for _, seg := range segments {
		last := len(groups) - 1
		if last >= 0 && seg.line-groups[last][len(groups[last])-1].line <= 1 {
			groups[last] = append(groups[last], seg)
			continue
		}
		groups = append(groups, []segment{seg})
	}
	return groups
}
