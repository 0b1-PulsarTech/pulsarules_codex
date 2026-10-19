package jsdoc

import "strings"

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
