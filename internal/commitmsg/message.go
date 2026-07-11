package commitmsg

import (
	"slices"
	"strings"
)

// Message is the structured representation of a parsed commit message.
// The parser fills these fields by consuming the raw message rune by rune.
type Message struct {
	// Emojis are the leading :shortcode: tokens (1-3 allowed).
	Emojis []string
	// Type is the Conventional Commit type (feat, fix, etc.).
	Type string
	// Scope is the optional parenthesized scope, without parens.
	Scope string
	// Breaking is true when the subject carries the ! marker.
	Breaking bool
	// Description is the subject text after "type(scope)!: ".
	Description string
	// Body is the optional text after the subject, separated by a blank line.
	Body string
	// Footers are the key-value trailers at the end (e.g. "Closes: #42").
	Footers []Footer
	// IsMerge is true for ":volcano: Merge ..." messages (no type required).
	IsMerge bool
	// IsInitial is true for ":ghost: Initial Commit" messages.
	IsInitial bool
	// IsWIP is true when the description starts with [wip] or [WIP].
	IsWIP bool
	// Raw is the original input message.
	Raw string
}

// Footer is a key-value trailer at the end of the commit body.
type Footer struct {
	// Key is the trailer token (e.g. "Closes", "Refs", "Co-Authored-By").
	Key string
	// Value is the trailer value text.
	Value string
}

// AllowedTypes are the Conventional Commit types recognized by the linter.
var AllowedTypes = []string{
	"feat", "fix", "refactor", "chore", "docs",
	"test", "perf", "build", "ci", "style", "revert",
}

// forbiddenTrailerKeys are trailer keys that mark a commit as tool-attributed;
// the commit rules reject any of them.
var forbiddenTrailerKeys = []string{"Co-Authored-By", "Claude-Session"}

// toolAttributionMarkers are lowercase substrings that betray an AI/tool
// attribution line even when it is not a well-formed trailer.
var toolAttributionMarkers = []string{
	"claude.ai/code",
	"noreply@anthropic.com",
	"generated with claude",
}

// ToolTrailer returns the first tool-attribution signal found in the message (a
// forbidden trailer key or an attribution marker anywhere in the raw text), or
// "" when the message carries none.
func (m Message) ToolTrailer() string {
	for _, f := range m.Footers {
		if slices.ContainsFunc(forbiddenTrailerKeys, func(key string) bool {
			return strings.EqualFold(f.Key, key)
		}) {
			return f.Key
		}
	}
	lowerRaw := strings.ToLower(m.Raw)
	for _, marker := range toolAttributionMarkers {
		if strings.Contains(lowerRaw, marker) {
			return marker
		}
	}
	return ""
}
