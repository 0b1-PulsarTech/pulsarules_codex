package commitmsg

import (
	"slices"
	"strings"
)

type Message struct {
	Emojis      []string
	Type        string
	Scope       string
	Breaking    bool
	Description string
	Body        string
	Footers     []Footer
	IsMerge     bool
	IsInitial   bool
	IsWIP       bool
	Raw         string
}

type Footer struct {
	Key   string
	Value string
}

func AllowedTypes() []string {
	return []string{
		"feat", "fix", "refactor", "chore", "docs",
		"test", "perf", "build", "ci", "style", "revert",
	}
}

func (msg Message) ToolTrailer() string {
	forbiddenTrailerKeys := []string{"Co-Authored-By", "Claude-Session"}
	toolAttributionMarkers := []string{
		"claude.ai/code",
		"noreply@anthropic.com",
		"generated with claude",
	}
	for _, footer := range msg.Footers {
		if slices.ContainsFunc(forbiddenTrailerKeys, func(key string) bool {
			return strings.EqualFold(footer.Key, key)
		}) {
			return footer.Key
		}
	}
	lowerRaw := strings.ToLower(msg.Raw)
	for _, marker := range toolAttributionMarkers {
		if strings.Contains(lowerRaw, marker) {
			return marker
		}
	}
	return ""
}
