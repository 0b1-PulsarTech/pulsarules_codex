package emoji

import (
	"fmt"
	"strings"
	"sync"
)

// theme ties the words a commit subject uses to the emoji family that names
// that area. It is what makes a suggestion relevant instead of merely valid:
// without it, ranking by usage alone offers whatever is popular, which reads
// as arbitrary next to the change being described.
type theme struct {
	keywords   []string
	shortcodes []string
}

// loadThemes parses the ordered theme table out of data/themes.txt, most
// specific area first, exactly once; every caller shares the cached slice (or
// the cached error). It is a sync.OnceValues rather than a package-level var
// initializer so a malformed file surfaces as a returned error on first use,
// not a boot-time panic or an empty table silently accepted.
var loadThemes = sync.OnceValues(func() ([]theme, error) {
	records, err := readRecords(dataFS, "data/themes.txt", 2)
	if err != nil {
		return nil, err
	}
	loaded := make([]theme, 0, len(records))
	for _, fields := range records {
		loaded = append(loaded, theme{
			keywords:   strings.Split(fields[0], ","),
			shortcodes: strings.Split(fields[1], ","),
		})
	}
	return loaded, nil
})

// Anchor names one theme's keyword area (its most distinctive keyword) and
// the emoji family that represents it in a commit subject.
type Anchor struct {
	Area       string
	Shortcodes []string
}

// Anchors returns the keyword-area to emoji-family table backing emoji
// selection, in the same specificity order themeMatches searches (most
// specific first). It is the single source the commits skill's emoji
// guidance renders from, so the prose and the matcher cannot drift apart. A
// malformed data/themes.txt is a build-time defect, so Anchors reports the
// error rather than rendering guidance from an empty table.
func Anchors() ([]Anchor, error) {
	themes, err := loadThemes()
	if err != nil {
		return nil, fmt.Errorf("build anchors: %w", err)
	}
	anchors := make([]Anchor, len(themes))
	for i, t := range themes {
		anchors[i] = Anchor{Area: t.keywords[0], Shortcodes: t.shortcodes}
	}
	return anchors, nil
}

// themeMatches returns the shortcodes whose area matches the words in text,
// most specific theme first. Matching is on whole-ish word prefixes so
// "refactoring" hits "refactor" and "tests" hits "test".
//
// simplification: Catalog.Suggest, themeMatches' only caller, has no error
// return in its contract, so a malformed data/themes.txt here degrades to no
// theme match rather than failing the suggestion outright. The same defect
// still surfaces loudly through Anchors and through NewCatalog's callers
// (validate, the commits skill render). Upgrade path: give Suggest an error
// return if a silent theme-match miss ever proves insufficient.
func themeMatches(text string) []string {
	themes, err := loadThemes()
	if err != nil {
		return nil
	}
	lowered := strings.ToLower(text)
	words := strings.FieldsFunc(lowered, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	if len(words) == 0 {
		return nil
	}

	var matched []string
	seen := make(map[string]bool)
	for _, candidate := range themes {
		if !hitsAny(words, candidate.keywords) {
			continue
		}
		for _, shortcode := range candidate.shortcodes {
			if !seen[shortcode] {
				seen[shortcode] = true
				matched = append(matched, shortcode)
			}
		}
	}
	return matched
}

func hitsAny(words, keywords []string) bool {
	for _, word := range words {
		for _, keyword := range keywords {
			if word == keyword || strings.HasPrefix(word, keyword) {
				return true
			}
		}
	}
	return false
}
