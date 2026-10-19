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

func footerKeys() []string {
	return append(
		toolFooterKeys(), "BREAKING CHANGE", "BREAKING-CHANGE", "Closes", "Refs",
	)
}

func isFooterLine(line string) bool {
	return slices.ContainsFunc(footerKeys(), func(key string) bool {
		return strings.HasPrefix(line, key+":")
	})
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
