package validate

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// emojiThemeShortcodes reports any commit emoji theme that names a shortcode
// absent from the commit emoji catalog: without this check, a theme could
// recommend an emoji the catalog then rejects at commit time, and the drift
// would only surface as a confusing hook failure instead of a validate error.
// A malformed catalog or theme table (a tools/emojigen build defect) is
// reported the same way, as a validate problem, rather than being swallowed.
func emojiThemeShortcodes(_ *knowledge.Index) []string {
	catalog, err := emoji.NewCatalog()
	if err != nil {
		return []string{fmt.Sprintf("load emoji catalog: %v", err)}
	}
	anchors, err := emoji.Anchors()
	if err != nil {
		return []string{fmt.Sprintf("load emoji anchors: %v", err)}
	}
	return themeShortcodesInCatalog(catalog, anchors)
}

// themeShortcodesInCatalog reports every anchor shortcode catalog does not
// allow, split out from emojiThemeShortcodes so a test can exercise the
// failing path with a fabricated anchor instead of only the embedded table.
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
