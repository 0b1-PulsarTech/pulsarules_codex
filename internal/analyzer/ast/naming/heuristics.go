package naming

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func isExported(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// maxCounterValue is the largest trailing number still read as a
// copy-paste counter; machine vocabulary (bit widths, hashes, utf8/sha256) starts at 8.
//
// simplification: sequences over 7 report only the first 7. Upgrade path:
// compare digit RUN LENGTH, require siblings to form a contiguous series.
const maxCounterValue = 7

// numberedStem splits a trailing digit run off name, reporting the stem and
// the run's VALUE. ok is false when there is no run, when the run is the whole
// name, or when the character before it is upper case - that marks an acronym
// or a version marker (H1, UTF8, V2) rather than a counter.
func numberedStem(name string) (stem string, value int, ok bool) {
	if name == "" {
		return "", 0, false
	}
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || trimmed == "" {
		return "", 0, false
	}
	lastLetter, _ := utf8.DecodeLastRuneInString(trimmed)
	if unicode.IsUpper(lastLetter) {
		return "", 0, false
	}

	parsed, err := strconv.Atoi(name[len(trimmed):])
	if err != nil {
		// A digit run too long to fit an int is not a counter either.
		return "", 0, false
	}
	return trimmed, parsed, true
}

// hungarianPrefixes are common type-encoding prefixes. They are matched
// case-sensitively against a name that starts lower case, because that is what
// Hungarian notation is: a lower-case type tag glued to a capitalised noun.
var hungarianPrefixes = []string{
	"lpstr", "lpsz", "psz", "str", "pfn", "ull",
	"dw", "sz", "ll", "ul", "us", "ui",
	"h", "u", "n", "b", "w", "l", "i",
}

// checkHungarian reports a lower-case type tag followed by a capitalised noun
// (strName, nCount). An identifier that STARTS upper case is never Hungarian:
// what matched there is the head of an acronym or of a MixedCaps word, which
// is how IDSelector, HTTPDoer, UUID and URLParser all used to be accused.
func checkHungarian(name string) bool {
	first, size := utf8.DecodeRuneInString(name)
	if size == 0 || !unicode.IsLower(first) {
		return false
	}
	for _, prefix := range hungarianPrefixes {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		next, nextSize := utf8.DecodeRuneInString(name[len(prefix):])
		if nextSize > 0 && unicode.IsUpper(next) {
			return true
		}
	}
	return false
}

// noiseWords carry no information beyond "a value". `base` is deliberately NOT
// here: unlike data or stuff, it names a real thing - the base of a
// calculation, a radix, a base URL, the embedded base type.
var noiseWords = map[string]bool{
	"data": true, "info": true, "temp": true, "tmp": true,
	"helper": true, "util": true, "misc": true,
	"stuff": true, "thing": true, "manager": true,
	"common": true, "dummy": true,
}

// simplification: only exact noise-word matches are flagged; prefix-contained
// noise words (e.g. dataValue) are considered specific enough. Upgrade path:
// switch to prefix detection if the false-positive rate proves acceptable.
func checkNoiseWord(name string) bool {
	return noiseWords[strings.ToLower(name)]
}
