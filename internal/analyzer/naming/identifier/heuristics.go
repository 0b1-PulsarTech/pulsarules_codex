package identifier

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func numberedStem(name string) (stem string, value int, ok bool) {
	if name == "" {
		return "", 0, false
	}
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || trimmed == "" {
		return "", 0, false
	}
	// An upper-case letter before the digits marks an acronym or version (H1, UTF8, V2).
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

func checkHungarian(name string) bool {
	first, size := utf8.DecodeRuneInString(name)
	// An upper-case start is an acronym or MixedCaps word head (IDSelector), not a tag.
	if size == 0 || !unicode.IsLower(first) {
		return false
	}
	// Hungarian notation is a lower-case type tag glued to a capitalised noun.
	hungarianPrefixes := []string{
		"lpstr", "lpsz", "psz", "str", "pfn", "ull",
		"dw", "sz", "ll", "ul", "us", "ui",
		"h", "u", "n", "b", "w", "l", "i",
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

func checkNoiseWord(name string) bool {
	// simplification: only exact noise-word matches are flagged; prefix-contained
	// noise words (e.g. dataValue) are considered specific enough. Upgrade path:
	// switch to prefix detection if the false-positive rate proves acceptable.
	// "base" is absent from the list because it names a real thing.
	switch strings.ToLower(name) {
	case "data", "info", "temp", "tmp", "helper", "util", "misc",
		"stuff", "thing", "manager", "common", "dummy":
		return true
	}
	return false
}
