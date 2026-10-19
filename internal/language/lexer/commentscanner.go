package lexer

import "strings"

type CommentScanner struct {
	syntax CommentSyntax
	open   *BlockDelim
	// depth counts the unclosed openers of a nested block; a flat block never passes 1.
	depth int
}

func NewCommentScanner(syntax CommentSyntax) *CommentScanner {
	return &CommentScanner{syntax: syntax}
}

func (scan *CommentScanner) InBlock() bool { return scan.open != nil }

func (scan *CommentScanner) Body(line string) (body string, inBlock bool) {
	inBlock = scan.InBlock()
	scan.scan(line)
	// Only a block's first line may open it, so "//" or "/*" on a continuation is text.
	if !inBlock {
		return CommentBody(line, scan.syntax), false
	}
	return tidy(trimClosers(strings.TrimSpace(line), scan.syntax), scan.syntax), true
}

type commentPiece struct {
	InBlock bool
	Rest    string
}

func (scan *CommentScanner) IsCommentLine(line string) bool {
	pieces, _ := scan.scan(line)
	return len(pieces) > 0
}

func (scan *CommentScanner) scan(line string) (pieces []commentPiece, hasCode bool) {
	rest := strings.TrimSpace(line)
	for rest != "" {
		if scan.open == nil {
			block, ok := scan.openingBlock(rest)
			if !ok {
				return scan.trailingLine(rest, pieces)
			}
			scan.open, scan.depth = &block, 1
			rest = strings.TrimSpace(rest[len(block.Open):])
		}
		pieces = append(pieces, commentPiece{InBlock: true, Rest: rest})
		end, depth := blockEnd(rest, *scan.open, scan.depth)
		scan.depth = depth
		if depth > 0 {
			return pieces, false
		}
		rest = strings.TrimSpace(rest[end:])
		scan.open = nil
	}
	return pieces, false
}

func (scan *CommentScanner) trailingLine(
	rest string, pieces []commentPiece,
) ([]commentPiece, bool) {
	for _, prefix := range scan.syntax.Line {
		if body, ok := strings.CutPrefix(rest, prefix); ok {
			return append(pieces, commentPiece{Rest: strings.TrimSpace(body)}), false
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

func blockEnd(rest string, block BlockDelim, depth int) (end, remaining int) {
	for end < len(rest) && depth > 0 {
		switch {
		case strings.HasPrefix(rest[end:], block.Close):
			depth--
			end += len(block.Close)
		case block.Nested && strings.HasPrefix(rest[end:], block.Open):
			depth++
			end += len(block.Open)
		default:
			end++
		}
	}
	return end, depth
}
