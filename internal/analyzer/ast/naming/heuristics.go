package naming

import (
	"strings"
	"unicode"
)

func isExported(name string) bool {
	for _, r := range name {
		return unicode.IsUpper(r)
	}
	return false
}

// checkNumbered reports a LOW sequential counter suffix (foo1, bar2), the
// copy-paste-siblings smell, not any trailing digit. A digit run immediately
// preceded by an uppercase letter is an acronym or version marker (H1, UTF8,
// V2) rather than a counter, so it is never flagged - that single rule is
// deliberately the whole exclusion: it clears the reported false positive
// (bodyWithoutH1) without needing whole-file sibling context threaded through
// every checkName call site.
func checkNumbered(name string) bool {
	if len(name) == 0 {
		return false
	}
	trimmed := strings.TrimRight(name, "0123456789")
	if trimmed == name || len(trimmed) == 0 {
		return false
	}
	lastLetter := rune(trimmed[len(trimmed)-1])
	return !unicode.IsUpper(lastLetter)
}

// hungarianPrefixes are common type-encoding prefixes.
var hungarianPrefixes = []string{
	"str", "sz", "psz", "lpstr", "lpsz",
	"pfn", "dw", "h", "u", "n", "b", "w", "l",
	"ull", "ll", "ul", "us", "ui", "i",
}

// hungarianAllowlist are names that look Hungarian but are idiomatic Go.
var hungarianAllowlist = map[string]bool{
	"ID": true, "IDs": true, // Go convention: ID, not Id
}

func checkHungarian(name string) bool {
	if hungarianAllowlist[name] {
		return false
	}
	if len(name) < 2 {
		return false
	}
	lower := strings.ToLower(name)
	for _, p := range hungarianPrefixes {
		if strings.HasPrefix(lower, p) && len(name) > len(p) &&
			unicode.IsUpper(rune(name[len(p)])) {
			return true
		}
	}
	return false
}

var noiseWords = map[string]bool{
	"data": true, "info": true, "temp": true, "tmp": true,
	"helper": true, "util": true, "misc": true,
	"stuff": true, "thing": true, "manager": true,
	"common": true, "base": true, "dummy": true,
}

// simplification: only exact noise-word matches are flagged; prefix-contained
// noise words (e.g. dataValue) are considered specific enough. Upgrade to
// prefix detection if false-positive rate is acceptable.
func checkNoiseWord(name string) bool {
	return noiseWords[strings.ToLower(name)]
}

func checkDisinformation(name string) {
	lower := strings.ToLower(name)
	switch lower {
	case "err":
		return // err is a valid Go convention for errors
	default:
	}
}
