package lexer

import "strings"

type CommentScanner struct {
	syntax CommentSyntax
	open   *BlockDelim
}

func NewCommentScanner(syntax CommentSyntax) *CommentScanner {
	return &CommentScanner{syntax: syntax}
}

func (scan *CommentScanner) InBlock() bool { return scan.open != nil }

type CommentPiece struct {
	InBlock bool
	Rest    string
}

func (scan *CommentScanner) IsCommentLine(line string) bool {
	pieces, _ := scan.Scan(line)
	return len(pieces) > 0
}

func (scan *CommentScanner) Scan(line string) (pieces []CommentPiece, hasCode bool) {
	rest := strings.TrimSpace(line)
	for rest != "" {
		if scan.open == nil {
			block, ok := scan.openingBlock(rest)
			if !ok {
				return scan.trailingLine(rest, pieces)
			}
			scan.open = &block
			rest = strings.TrimSpace(rest[len(block.Open):])
		}
		pieces = append(pieces, CommentPiece{InBlock: true, Rest: rest})
		closeIdx := strings.Index(rest, scan.open.Close)
		if closeIdx == -1 {
			return pieces, false
		}
		rest = strings.TrimSpace(rest[closeIdx+len(scan.open.Close):])
		scan.open = nil
	}
	return pieces, false
}

func (scan *CommentScanner) trailingLine(
	rest string, pieces []CommentPiece,
) ([]CommentPiece, bool) {
	for _, prefix := range scan.syntax.Line {
		if body, ok := strings.CutPrefix(rest, prefix); ok {
			return append(pieces, CommentPiece{Rest: strings.TrimSpace(body)}), false
		}
	}
	return pieces, true
}

func (scan *CommentScanner) openingBlock(rest string) (BlockDelim, bool) {
	// Checked before line prefixes: Lua's "--[[" would read as a "--" line.
	for _, block := range scan.syntax.Blocks {
		if strings.HasPrefix(rest, block.Open) {
			return block, true
		}
	}
	return BlockDelim{}, false
}
