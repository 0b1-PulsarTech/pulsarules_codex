package pathalias

import "bytes"

const maxConfigBytes = 1 << 20

func stripJSONC(body []byte) ([]byte, bool) {
	if len(body) > maxConfigBytes {
		return nil, false
	}
	return dropTrailingCommas(stripComments(body)), true
}

func stripComments(body []byte) []byte {
	out := make([]byte, 0, len(body))
	for pos := 0; pos < len(body); {
		switch {
		case body[pos] == '"':
			end := stringEnd(body, pos)
			out = append(out, body[pos:end]...)
			pos = end
		case bytes.HasPrefix(body[pos:], []byte("//")):
			pos = lineEnd(body, pos)
		case bytes.HasPrefix(body[pos:], []byte("/*")):
			out = append(out, ' ')
			pos = blockEnd(body, pos)
		default:
			out = append(out, body[pos])
			pos++
		}
	}
	return out
}

func dropTrailingCommas(body []byte) []byte {
	out := make([]byte, 0, len(body))
	for pos := 0; pos < len(body); {
		if body[pos] == '"' {
			end := stringEnd(body, pos)
			out = append(out, body[pos:end]...)
			pos = end
			continue
		}
		if body[pos] == ',' && closesNext(body, pos+1) {
			pos++
			continue
		}
		out = append(out, body[pos])
		pos++
	}
	return out
}

func closesNext(body []byte, pos int) bool {
	for ; pos < len(body); pos++ {
		switch body[pos] {
		case ' ', '\t', '\r', '\n':
		case '}', ']':
			return true
		default:
			return false
		}
	}
	return false
}

func stringEnd(body []byte, start int) int {
	for pos := start + 1; pos < len(body); pos++ {
		switch body[pos] {
		case '\\':
			pos++
		case '"':
			return pos + 1
		}
	}
	return len(body)
}

func lineEnd(body []byte, start int) int {
	if end := bytes.IndexByte(body[start:], '\n'); end >= 0 {
		return start + end
	}
	return len(body)
}

func blockEnd(body []byte, start int) int {
	if end := bytes.Index(body[start+2:], []byte("*/")); end >= 0 {
		return start + 2 + end + 2
	}
	return len(body)
}
