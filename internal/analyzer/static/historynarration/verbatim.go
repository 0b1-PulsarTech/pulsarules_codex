package historynarration

import "strings"

func verbatimBefore(line string, at int) bool {
	start := at
	for start > 0 && (line[start-1] == '@' || line[start-1] == '$') {
		start--
	}
	return strings.Contains(line[start:at], "@")
}

func verbatimEnd(line string, at int) (end int, closed bool) {
	for pos := at + 1; pos < len(line); pos++ {
		if line[pos] != '"' {
			continue
		}
		if pos+1 < len(line) && line[pos+1] == '"' {
			pos++
			continue
		}
		return pos + 1, true
	}
	return len(line), false
}
