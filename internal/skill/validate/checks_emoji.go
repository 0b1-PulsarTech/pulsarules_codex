package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func emojiThemeShortcodes(_ *knowledge.Index) []string {
	catalog, err := emoji.NewCatalog()
	if err != nil {
		return []string{fmt.Sprintf("load emoji catalog: %v", err)}
	}
	anchors, err := emoji.Anchors()
	if err != nil {
		return []string{fmt.Sprintf("load emoji anchors: %v", err)}
	}
	// A bad theme surfaces as a validate error, not a confusing hook failure at commit time.
	return themeShortcodesInCatalog(catalog, anchors)
}

func themeShortcodesInCatalog(catalog *emoji.Catalog, anchors []emoji.Anchor) []string {
	var problems []string
	for _, anchor := range anchors {
		for _, shortcode := range anchor.Shortcodes {
			if !catalog.Allows(shortcode) {
				problems = append(problems, fmt.Sprintf(
					"emoji theme %q lists off-catalog shortcode %q", anchor.Area, shortcode,
				))
			}
		}
	}
	return problems
}
