package blocksize

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type limit struct {
	maxLines int
	reporter finding.Reporter
}

func scanFile(
	src []byte, fc pass.FileChange, syntax lexer.CommentSyntax, bound limit,
) []finding.Finding {
	var findings []finding.Finding
	scanner := lexer.NewCommentScanner(syntax)
	blockStart := 0
	blockLines := 0
	exempt := false
	flush := func() {
		if blockLines > bound.maxLines && !exempt {
			findings = append(findings, bound.reporter.At(
				fc.Path,
				blockStart,
				"comment block is "+strconv.Itoa(blockLines)+" lines, max "+
					strconv.Itoa(bound.maxLines),
				"split the comment or move the explanation into named functions/types",
			))
		}
		blockLines = 0
		exempt = false
	}
	lines := strings.Split(string(src), "\n")
	for idx := 0; idx < len(lines); idx++ {
		line := lines[idx]
		if span := directiveSpan(scanner, lines[idx:], syntax); span > 1 {
			idx += span - 1
			continue
		}
		if !scanner.IsCommentLine(line) {
			flush()
			continue
		}
		if lexer.IsMachineReadCommentLine(line) && !opensHandBlock(scanner, lines[idx:]) {
			continue
		}
		if blockLines == 0 {
			blockStart = idx + 1
			// A simplification block states the ceiling and an upgrade path, which no
			// 3-line cap holds.
			exempt = lexer.IsGeneratedMarkerLine(line) ||
				lexer.IsSimplificationMarker(line, syntax)
		}
		blockLines++
	}
	flush()
	return findings
}

func directiveSpan(
	scanner *lexer.CommentScanner,
	lines []string,
	syntax lexer.CommentSyntax,
) int {
	if scanner.InBlock() {
		return 0
	}
	span := lexer.ExemptHeaderSpan(lines)
	if span <= 1 {
		return span
	}
	probe := lexer.NewCommentScanner(syntax)
	// The opener must open a real block in this language, or its text is a string or heredoc.
	probe.IsCommentLine(lines[0])
	if !probe.InBlock() {
		return 0
	}
	for _, line := range lines[1:span] {
		probe.IsCommentLine(line)
	}
	if probe.InBlock() {
		return 0
	}
	return span
}

func opensHandBlock(scanner *lexer.CommentScanner, lines []string) bool {
	if !scanner.InBlock() {
		return false
	}
	trimmed := strings.TrimSpace(lines[0])
	for _, delim := range lexer.AnyCommentSyntax().Blocks {
		if rest, ok := strings.CutPrefix(trimmed, delim.Open); ok {
			_, after, closed := strings.Cut(rest, delim.Close)
			remainder := append([]string{strings.TrimSpace(after)}, lines[1:]...)
			return closed && lexer.ExemptHeaderSpan(remainder) == 0
		}
	}
	return false
}
