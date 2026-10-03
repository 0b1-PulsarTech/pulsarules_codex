package bigcomment

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/lexical"
)

type limit struct {
	maxLines int
	reporter core.Reporter
}

func scanFile(
	src []byte, fc core.FileChange, syntax lexical.CommentSyntax, bound limit,
) []core.Finding {
	var findings []core.Finding
	scanner := lexical.NewCommentScanner(syntax)
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
		if lexical.IsMachineReadCommentLine(line) && !opensHandBlock(scanner, lines[idx:]) {
			continue
		}
		if blockLines == 0 {
			blockStart = idx + 1
			// A simplification block states the ceiling and an upgrade path, which no
			// 3-line cap holds.
			exempt = lexical.IsGeneratedMarkerLine(line) || lexical.IsSimplificationMarker(line)
		}
		blockLines++
	}
	flush()
	return findings
}

func directiveSpan(
	scanner *lexical.CommentScanner,
	lines []string,
	syntax lexical.CommentSyntax,
) int {
	if scanner.InBlock() {
		return 0
	}
	span := lexical.ExemptHeaderSpan(lines)
	if span <= 1 {
		return span
	}
	probe := lexical.NewCommentScanner(syntax)
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

func opensHandBlock(scanner *lexical.CommentScanner, lines []string) bool {
	if !scanner.InBlock() {
		return false
	}
	trimmed := strings.TrimSpace(lines[0])
	for _, delims := range [][2]string{{"/*", "*/"}, {"<!--", "-->"}} {
		if rest, ok := strings.CutPrefix(trimmed, delims[0]); ok {
			_, after, closed := strings.Cut(rest, delims[1])
			remainder := append([]string{strings.TrimSpace(after)}, lines[1:]...)
			return closed && lexical.ExemptHeaderSpan(remainder) == 0
		}
	}
	return false
}
