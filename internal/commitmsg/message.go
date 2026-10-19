package commitmsg

import (
	"slices"
	"strings"
)

type Kind int

const (
	Regular Kind = iota
	Initial
	Merge
)

const (
	InitialEmoji = "ghost"
	MergeEmoji   = "volcano"
)

type Message struct {
	Kind          Kind
	LeadingEmojis []string
	Type          string
	Scope         string
	Breaking      bool
	Description   string
	Body          string
	Footers       []Footer
	IsWIP         bool
	Raw           string
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

func toolFooterKeys() []string {
	return []string{"Co-Authored-By", "Claude-Session"}
}

func (msg Message) IsCanonicalSpecial() bool {
	switch msg.Kind {
	case Initial:
		return len(msg.LeadingEmojis) == 1 && msg.LeadingEmojis[0] == InitialEmoji
	case Merge:
		return len(msg.LeadingEmojis) >= 1 && msg.LeadingEmojis[0] == MergeEmoji
	case Regular:
		return false
	}
	return false
}

func (msg Message) ToolTrailer() string {
	toolAttributionMarkers := []string{
		"claude.ai/code",
		"noreply@anthropic.com",
		"generated with claude",
	}
	for _, footer := range msg.Footers {
		if slices.ContainsFunc(toolFooterKeys(), func(key string) bool {
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
