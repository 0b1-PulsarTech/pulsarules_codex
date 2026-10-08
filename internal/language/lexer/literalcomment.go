package lexer

import "strings"

func (scan *literalScan) commentLen(rest string) int {
	// Blocks first: a block opener may start with a line opener's prefix.
	for _, block := range scan.comments.Blocks {
		if strings.HasPrefix(rest, block.Open) {
			return scan.blockLen(rest, block)
		}
	}
	for _, prefix := range scan.comments.Line {
		if strings.HasPrefix(rest, prefix) {
			line, _, _ := strings.Cut(rest, "\n")
			scan.work += len(line)
			return len(line)
		}
	}
	return 0
}

func (scan *literalScan) blockLen(rest string, block BlockDelim) int {
	depth, at := 1, len(block.Open)
	for at < len(rest) && depth > 0 {
		switch {
		case strings.HasPrefix(rest[at:], block.Close):
			depth--
			at += len(block.Close)
		case block.Nested && strings.HasPrefix(rest[at:], block.Open):
			depth++
			at += len(block.Open)
		default:
			at++
		}
	}
	scan.work += at
	// A block still open at EOF swallows the rest of the file.
	if depth > 0 {
		scan.cut(cutAtEOF)
	}
	return at
}
