package emoji

import (
	"io/fs"
	"strings"
)

type theme struct {
	keywords   []string
	shortcodes []string
}

func loadThemes(fsys fs.FS) ([]theme, error) {
	records, err := readRecords(fsys, "data/themes.txt", 2)
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
}

type Anchor struct {
	Area       string
	Shortcodes []string
}

func (catalog *Catalog) Anchors() []Anchor {
	anchors := make([]Anchor, len(catalog.themes))
	for idx, theme := range catalog.themes {
		anchors[idx] = Anchor{Area: theme.keywords[0], Shortcodes: theme.shortcodes}
	}
	return anchors
}

func (catalog *Catalog) themeMatches(text string) []string {
	lowered := strings.ToLower(text)
	words := strings.FieldsFunc(lowered, func(char rune) bool {
		return (char < 'a' || char > 'z') && (char < '0' || char > '9')
	})
	if len(words) == 0 {
		return nil
	}

	var matched []string
	seen := make(map[string]bool)
	for _, candidate := range catalog.themes {
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
