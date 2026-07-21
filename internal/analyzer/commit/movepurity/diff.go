package movepurity

import (
	"strings"
	"unicode"
)

type diffReader interface {
	StagedDiff(path string) (string, error)
	StagedRenameDiff(oldPath, newPath string, minScore int) (string, error)
}

func isImportOnlyDiff(diff string) bool {
	for line := range strings.SplitSeq(diff, "\n") {
		content, ok := diffContentLine(line)
		if !ok {
			continue
		}
		if !isImportOnlyLine(content) {
			return false
		}
	}
	return true
}

func diffContentLine(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		return "", false
	case strings.HasPrefix(line, "+"), strings.HasPrefix(line, "-"):
		return line[1:], true
	default:
		return "", false
	}
}

func isImportOnlyLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	switch {
	case trimmed == "", trimmed == "(", trimmed == ")", trimmed == "import (":
		return true
	case strings.HasPrefix(trimmed, "package "):
		return true
	case strings.HasPrefix(trimmed, "import "):
		return isImportSpec(strings.TrimPrefix(trimmed, "import "))
	default:
		return isImportSpec(trimmed)
	}
}

func isImportSpec(spec string) bool {
	if isQuoted(spec) {
		return true
	}
	alias, path, ok := strings.Cut(spec, " ")
	if !ok {
		return false
	}
	return isGoIdentifier(alias) && isQuoted(strings.TrimSpace(path))
}

func isQuoted(s string) bool {
	return len(s) >= 2 && strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`)
}

func isGoIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || unicode.IsLetter(r):
		case i > 0 && unicode.IsDigit(r):
		default:
			return false
		}
	}
	return true
}
