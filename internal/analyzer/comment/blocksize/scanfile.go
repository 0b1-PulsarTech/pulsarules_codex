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
	lineNum := 0
	exempt := false
	for line := range strings.SplitSeq(string(src), "\n") {
		lineNum++
		if scanner.IsCommentLine(line) {
			if pass.IsDirectiveHeaderLine(line) {
				continue
			}
			if blockLines == 0 {
				blockStart = lineNum
				// A simplification block states the ceiling and an upgrade path, which no
				// 3-line cap holds.
				exempt = pass.IsGeneratedMarkerLine(line) || pass.IsSimplificationMarker(line)
			}
			blockLines++
			continue
		}
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
	if blockLines > maxLines && !exempt {
		findings = append(findings, reporter.At(
			fc.Path,
			blockStart,
			"comment block is "+strconv.Itoa(blockLines)+" lines, max "+strconv.Itoa(maxLines),
			"split the comment or move the explanation into named functions/types",
		))
	}
	return findings
}
