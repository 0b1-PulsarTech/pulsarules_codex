package movepurity

import (
	"strings"
	"unicode"
)

// diffReader is the git access this analyzer needs to inspect what a
// staged change contains, so it can tell a pure import-path fixup from a
// real content edit.
type diffReader interface {
	StagedDiff(path string) (string, error)
	StagedRenameDiff(oldPath, newPath string, minScore int) (string, error)
}

// why: a line this cannot confidently classify counts as an edit (returns
// false), so a real change is never mistaken for a pure move; a diff with
// no content lines vacuously counts as import-only.
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

// why: only the mechanical consequences of a move (package clause,
// import-block punctuation, a bare or aliased import path) are allowed;
// anything else is a real edit.
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

// isImportSpec: a bare import path ("pkg/path") or an aliased one (alias
// "pkg/path", including the blank identifier "_").
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
