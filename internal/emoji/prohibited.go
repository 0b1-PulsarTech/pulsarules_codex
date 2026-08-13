package emoji

import "slices"

// prohibited maps a banned shortcode to its sanctioned replacement, empty when
// there is no single substitute. These are banned by decision, not by
// rendering: they are excluded from the catalog even when the reference
// repository uses them.
var prohibited = map[string]string{
	"robot":     "",
	"test_tube": "tea",
	"compass":   "",
	"sparkles":  "",
}

// nonRendering lists shortcodes that appear as raw text instead of an emoji in
// the owner's GitKraken, confirmed by hand. All three are Unicode 2016 or
// later; the version cutoff in the generator already excludes them, and they
// are named here so the reason survives a catalog regeneration. Add to this
// list if raw text shows up again.
var nonRendering = []string{"clown_face", "adhesive_bandage", "abacus"}

func IsProhibited(shortcode string) bool {
	_, found := prohibited[shortcode]
	return found
}

// ProhibitedReplacement returns the sanctioned substitute for a banned
// shortcode. The second result reports whether the shortcode is banned at all,
// distinguishing "banned, no substitute" from "not banned".
func ProhibitedReplacement(shortcode string) (string, bool) {
	replacement, found := prohibited[shortcode]
	return replacement, found
}

func IsNonRendering(shortcode string) bool {
	return slices.Contains(nonRendering, shortcode)
}
