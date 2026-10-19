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
		if strings.HasPrefix(rest, prefix) && scan.lineMayOpen() {
			line, _, _ := strings.Cut(rest, "\n")
			scan.work += len(line)
			return len(line)
		}
	}
	return 0
}

func (scan *literalScan) blockLen(rest string, block BlockDelim) int {
	end, depth := blockEnd(rest[len(block.Open):], block, 1)
	at := len(block.Open) + end
	scan.work += at
	// A block still open at EOF swallows the rest of the file.
	if depth > 0 {
		scan.cut(cutAtEOF)
	}
	return at
}

func (scan *literalScan) lineMayOpen() bool {
	return scan.comments.LineOpens == nil || scan.comments.LineOpens(scan.src, scan.pos)
}
