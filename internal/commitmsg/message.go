package commitmsg

import "strings"

type Kind int

const (
	Regular Kind = iota
	Initial
	Merge
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

func (msg Message) ToolTrailer() string {
	toolAttributionMarkers := []string{
		"claude.ai/code",
		"noreply@anthropic.com",
		"generated with claude",
	}
	for _, footer := range msg.Footers {
		if isToolFooterKey(footer.Key) {
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
