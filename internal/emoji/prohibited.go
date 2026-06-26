package emoji

func IsProhibited(shortcode string) bool {
	_, found := ProhibitedReplacement(shortcode)
	return found
}

func ProhibitedReplacement(shortcode string) (string, bool) {
	switch shortcode {
	// Banned by decision, not by rendering, even when the reference repo uses them.
	case "test_tube":
		return "tea", true
	case "robot", "compass", "sparkles":
		return "", true
	}
	return "", false
}

func IsNonRendering(shortcode string) bool {
	switch shortcode {
	// Named here so the reason survives regeneration: these show as raw text in the
	// owner's GitKraken (confirmed by hand).
	case "clown_face", "adhesive_bandage", "abacus":
		return true
	}
	return false
}
