package narration

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func scanSegments(src, ext string) []segment {
	lang := language.ForExtension(ext)
	var segments []segment
	line, from := 1, 0
	for _, span := range language.ScanComments(ext, src) {
		line += strings.Count(src[from:span.Start], "\n")
		from = span.Start
		for _, part := range commentParts(src[span.Start:span.End], lang.Comments) {
			segments = append(segments, segment{line: line + part.row, text: part.text})
		}
	}
	return segments
}

type part struct {
	row  int
	text string
}

func commentParts(text string, syntax lexer.CommentSyntax) []part {
	for _, block := range syntax.Blocks {
		if rest, found := strings.CutPrefix(text, block.Open); found {
			return blockParts(rest, block.Close)
		}
	}
	for _, prefix := range syntax.Line {
		if rest, found := strings.CutPrefix(text, prefix); found {
			return []part{{text: rest}}
		}
	}
	return nil
}

func blockParts(rest, closer string) []part {
	var parts []part
	lines := strings.Split(rest, "\n")
	for row, text := range lines {
		// An empty line of a block comment says nothing, so it leaves a gap in the block.
		if text == "" {
			continue
		}
		if row == len(lines)-1 {
			text = strings.TrimSuffix(text, closer)
		}
		parts = append(parts, part{row: row, text: text})
	}
	return parts
}
