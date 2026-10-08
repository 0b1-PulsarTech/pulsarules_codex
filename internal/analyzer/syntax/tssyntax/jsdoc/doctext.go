package jsdoc

import (
	"slices"
	"strings"
)

const maxDeprecatedLines = 3

func docLines(comment string) []string {
	body := strings.TrimSuffix(strings.TrimPrefix(comment, "/**"), "*/")
	lines := strings.Split(body, "\n")
	for idx, line := range lines {
		lines[idx] = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "*"))
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func commentBody(comment string) string {
	body := strings.TrimPrefix(comment, "//")
	if body == comment {
		body = strings.TrimSuffix(strings.TrimPrefix(comment, "/*"), "*/")
	}
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(body), "*"))
}

func isDirective(line string) bool {
	for _, prefix := range []string{
		"eslint-", "@ts-", "pulsarules:", "ignore:", "ignore_for_file:", "prettier-ignore",
		"istanbul ignore", "c8 ignore", "svelte-ignore", "biome-ignore", "noinspection",
	} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

func isDeprecatedOnly(prose, own []string) bool {
	marked := strings.HasPrefix(prose[0], "@deprecated") ||
		strings.HasPrefix(prose[0], "Deprecated:")
	return marked && len(own) <= maxDeprecatedLines && !slices.Contains(own, "")
}

func isSimplification(prose []string) bool {
	return strings.HasPrefix(prose[0], "simplification:")
}

func isTagsOnly(prose []string) bool {
	for _, line := range prose {
		if !isTypeTag(line) {
			return false
		}
	}
	return true
}

func isTypeTag(line string) bool {
	if !strings.HasPrefix(line, "@") {
		return false
	}
	end := 1
	for end < len(line) && isLetter(line[end]) {
		end++
	}
	rest := strings.TrimSpace(line[end:])
	switch line[:end] {
	case "@import":
		return true
	case "@type", "@returns", "@satisfies":
		return typeOnly(rest)
	case "@param", "@typedef", "@template", "@property", "@callback":
		return typeAndName(rest)
	}
	return false
}

func typeOnly(rest string) bool {
	left, ok := afterType(rest)
	return ok && left == ""
}

func typeAndName(rest string) bool {
	left, ok := afterType(rest)
	return ok && !strings.ContainsAny(left, " \t")
}

func afterType(rest string) (string, bool) {
	if !strings.HasPrefix(rest, "{") {
		return rest, true
	}
	length := bracedLength(rest)
	return strings.TrimSpace(rest[length:]), length > 0
}

func bracedLength(text string) int {
	depth := 0
	for idx := range len(text) {
		switch text[idx] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return idx + 1
			}
		}
	}
	return 0
}

func isLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}
