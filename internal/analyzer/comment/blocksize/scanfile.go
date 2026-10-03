package blocksize

import (
	"strconv"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func scanFile(
	src []byte,
	fc pass.FileChange,
	syntax pass.CommentSyntax,
	maxLines int,
	reporter pass.Reporter,
) []pass.Finding {
	var findings []pass.Finding
	scanner := pass.NewCommentScanner(syntax)
	blockStart := 0
	blockLines := 0
	exempt := false
	flush := func() {
		if blockLines > maxLines && !exempt {
			findings = append(findings, reporter.At(
				fc.Path,
				blockStart,
				"comment block is "+strconv.Itoa(blockLines)+" lines, max "+strconv.Itoa(maxLines),
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
		if pass.IsMachineReadCommentLine(line) && !opensHandBlock(scanner, lines[idx:]) {
			continue
		}
		if blockLines == 0 {
			blockStart = idx + 1
			// A simplification block states the ceiling and an upgrade path, which no
			// 3-line cap holds.
			exempt = pass.IsGeneratedMarkerLine(line) || pass.IsSimplificationMarker(line)
		}
		blockLines++
	}
	flush()
	return findings
}

func directiveSpan(
	scanner *pass.CommentScanner,
	lines []string,
	syntax pass.CommentSyntax,
) int {
	if scanner.InBlock() {
		return 0
	}
	span := pass.ExemptHeaderSpan(lines)
	if span <= 1 {
		return span
	}
	probe := pass.NewCommentScanner(syntax)
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

func opensHandBlock(scanner *pass.CommentScanner, lines []string) bool {
	if !scanner.InBlock() {
		return false
	}
	trimmed := strings.TrimSpace(lines[0])
	for _, delims := range [][2]string{{"/*", "*/"}, {"<!--", "-->"}} {
		if rest, ok := strings.CutPrefix(trimmed, delims[0]); ok {
			_, after, closed := strings.Cut(rest, delims[1])
			remainder := append([]string{strings.TrimSpace(after)}, lines[1:]...)
			return closed && pass.ExemptHeaderSpan(remainder) == 0
		}
	}
	return false
}
