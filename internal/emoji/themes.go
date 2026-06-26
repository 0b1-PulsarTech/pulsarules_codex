package emoji

import (
	"fmt"
	"strings"
)

type theme struct {
	keywords   []string
	shortcodes []string
}

func loadThemes() ([]theme, error) {
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
}

type Anchor struct {
	Area       string
	Shortcodes []string
}

func Anchors() ([]Anchor, error) {
	themes, err := loadThemes()
	if err != nil {
		return nil, fmt.Errorf("build anchors: %w", err)
	}
	anchors := make([]Anchor, len(themes))
	for idx, theme := range themes {
		anchors[idx] = Anchor{Area: theme.keywords[0], Shortcodes: theme.shortcodes}
	}
	return anchors, nil
}
