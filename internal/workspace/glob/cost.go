package glob

import "strings"

func HasBrace(pattern string) bool {
	// A brace inside a [...] class is a literal member, so only those outside one count.
	inClass := false
	for idx := 0; idx < len(pattern); idx++ {
		switch char := pattern[idx]; {
		case char == '\\':
			idx++
		case inClass:
			inClass = char != ']'
		case char == '[':
			inClass = true
		case char == '{' || char == '}':
			return true
		}
	}
	return false
}

func StarCount(pattern string) int {
	count := 0
	for part := range strings.SplitSeq(pattern, "/") {
		if part != "**" && part != "*" {
			count += strings.Count(part, "*")
		}
	}
	return count
}

func WildcardWork(pattern string) int {
	// Each ? and each byte of a [...] class (brackets included) costs path.Match a re-scan.
	work, inClass := 0, false
	for idx := 0; idx < len(pattern); idx++ {
		char := pattern[idx]
		switch {
		case char == '\\':
			idx++
			if inClass {
				work += 2
			}
		case inClass:
			work++
			inClass = char != ']'
		case char == '[':
			work++
			inClass = true
		case char == '?':
			work++
		}
	}
	return work
}

func StarTail(pattern string) int {
	// path.Match re-scans the rest of a segment from every start position of a *.
	tail := 0
	for part := range strings.SplitSeq(pattern, "/") {
		if star := strings.IndexByte(part, '*'); star >= 0 {
			tail += len(part) - star - strings.Count(part[star:], "*")
		}
	}
	return tail
}
