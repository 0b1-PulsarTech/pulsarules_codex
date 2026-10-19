package commitmsg

import (
	"slices"
	"strings"
)

func kindOf(text string) Kind {
	if strings.HasPrefix(text, "Initial Commit") || strings.HasPrefix(text, "Initial commit") {
		return Initial
	}
	if strings.HasPrefix(text, "Merge ") {
		return Merge
	}
	return Regular
}

func isToolFooterKey(key string) bool {
	return slices.ContainsFunc(toolFooterKeys(), func(toolKey string) bool {
		return strings.EqualFold(key, toolKey)
	})
}

func isFooterLine(line string) bool {
	if key, _, found := strings.Cut(line, ":"); found && isToolFooterKey(key) {
		return true
	}
	// Only tool keys fold: Closes/Refs folding would change Body findings, BREAKING stays upper.
	return slices.ContainsFunc(
		[]string{"BREAKING CHANGE", "BREAKING-CHANGE", "Closes", "Refs"},
		func(key string) bool { return strings.HasPrefix(line, key+":") },
	)
}

func parseFooter(line string) Footer {
	for _, sep := range []string{": ", " #"} {
		if idx := strings.Index(line, sep); idx > 0 {
			return Footer{
				Key:   strings.TrimSpace(line[:idx]),
				Value: strings.TrimSpace(line[idx+len(sep):]),
			}
		}
	}
	return Footer{}
}
