package narration

import "strings"

func (sc *scanner) regionEnd(rest string) int {
	for _, tag := range []string{"</script", "</style"} {
		if strings.HasPrefix(rest, tag) && tagBoundary(rest, len(tag)) {
			return len(tag)
		}
	}
	return 0
}

func tagBoundary(rest string, size int) bool {
	if size >= len(rest) {
		return true
	}
	return strings.IndexByte(">/ \t\r", rest[size]) >= 0
}

func indexTag(rest, open string) int {
	for from := 0; from < len(rest); {
		at := strings.Index(rest[from:], open)
		if at < 0 {
			return -1
		}
		if open == "<!--" || tagBoundary(rest[from+at:], len(open)) {
			return from + at
		}
		from += at + 1
	}
	return -1
}

func (sc *scanner) markupStep(line string, pos int) int {
	rest := line[pos:]
	best, kind := -1, ""
	for _, open := range []string{"<!--", "<script", "<style"} {
		at := indexTag(rest, open)
		if at >= 0 && (best < 0 || at < best) {
			best, kind = at, open
		}
	}
	switch {
	case best < 0:
		return len(line)
	case kind == "<!--":
		sc.closer = "-->"
		return pos + best + len(kind)
	}
	sc.region = regionScript
	if kind == "<style" {
		sc.region = regionStyle
	}
	tagEnd := strings.IndexByte(rest[best:], '>')
	if tagEnd < 0 {
		return len(line)
	}
	return pos + best + tagEnd + 1
}
