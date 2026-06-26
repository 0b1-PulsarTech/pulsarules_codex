package knowledge

import "strings"

// FirstSentence returns text up to and including the first period, or text
// unchanged when it has none. It shortens a description to a single sentence for
// listings and the router's available-skills summary.
func FirstSentence(text string) string {
	if i := strings.IndexByte(text, '.'); i >= 0 {
		return text[:i+1]
	}
	return text
}
