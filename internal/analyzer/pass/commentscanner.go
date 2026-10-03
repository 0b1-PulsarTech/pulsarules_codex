package pass

import "strings"

type CommentScanner struct {
	syntax CommentSyntax
	open   *BlockDelim
}

func NewCommentScanner(syntax CommentSyntax) *CommentScanner {
	return &CommentScanner{syntax: syntax}
}

func (scan *CommentScanner) InBlock() bool { return scan.open != nil }

func (scan *CommentScanner) IsCommentLine(line string) bool {
	rest := strings.TrimSpace(line)
	sawComment := false
	for rest != "" {
		if scan.open != nil {
			sawComment = true
			closeIdx := strings.Index(rest, scan.open.Close)
			if closeIdx == -1 {
				return true
			}
			rest = strings.TrimSpace(rest[closeIdx+len(scan.open.Close):])
			scan.open = nil
			continue
		}
		if block, ok := scan.openingBlock(rest); ok {
			sawComment = true
			rest = strings.TrimSpace(rest[len(block.Open):])
			scan.open = &block
			continue
		}
		if scan.hasLinePrefix(rest) {
			return true
		}
		break
	}
	return sawComment
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

func (scan *CommentScanner) hasLinePrefix(rest string) bool {
	for _, prefix := range scan.syntax.Line {
		if strings.HasPrefix(rest, prefix) {
			return true
		}
	}
	return false
}
