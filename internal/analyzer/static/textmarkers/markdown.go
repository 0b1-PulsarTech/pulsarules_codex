package textmarkers

import "strings"

// region is a half-open byte range [start, end) of src.
type region struct {
	start int
	end   int
}

// codeRegions returns the byte ranges markdown renders as code: fenced blocks
// and inline spans.
// why: a markdown fence is unambiguous, unlike a Go string literal, so the
// character inside one is content being shown rather than prose style.

// simplification: a fence is a run of 3+ backticks or tildes closed by an
// equal-or-longer run of the same char; an inline span is a backtick run
// closed by an equal run on its line. Upgrade path: a CommonMark scanner if
// nested or escaped delimiters start giving wrong answers.
func codeRegions(src string) []region {
	var regions []region
	offset := 0
	fenceChar := byte(0)
	fenceStart := 0
	fenceLen := 0

	for _, line := range strings.SplitAfter(src, "\n") {
		if char, run := fenceDelimiter(line); run >= minFenceRun {
			switch {
			case fenceChar == 0:
				fenceChar, fenceLen, fenceStart = char, run, offset
			case char == fenceChar && run >= fenceLen:
				regions = append(regions, region{start: fenceStart, end: offset + len(line)})
				fenceChar, fenceLen = 0, 0
			}
		} else if fenceChar == 0 {
			regions = append(regions, inlineSpans(line, offset)...)
		}
		offset += len(line)
	}
	// An unclosed fence runs to the end of the file, the way a renderer treats it.
	if fenceChar != 0 {
		regions = append(regions, region{start: fenceStart, end: len(src)})
	}
	return regions
}

// minFenceRun is the delimiter length markdown requires to open a code fence.
const minFenceRun = 3

// fenceDelimiter reports the fence character a line opens or closes with and
// how long its run is, or a zero char when the line is not a fence.
func fenceDelimiter(line string) (byte, int) {
	trimmed := strings.TrimLeft(line, " ")
	if trimmed == "" {
		return 0, 0
	}
	char := trimmed[0]
	if char != '`' && char != '~' {
		return 0, 0
	}
	run := 0
	for run < len(trimmed) && trimmed[run] == char {
		run++
	}
	return char, run
}

// inlineSpans returns the code-span ranges within one line, offset by base.
func inlineSpans(line string, base int) []region {
	var spans []region
	for i := 0; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		open := i
		for i < len(line) && line[i] == '`' {
			i++
		}
		run := i - open
		if end := closingRun(line, i, run); end >= 0 {
			spans = append(spans, region{start: base + open, end: base + end + run})
			i = end + run
		}
	}
	return spans
}

// closingRun returns the index of the backtick run of exactly length run that
// closes a span opened before from, or -1 when the line has none.
func closingRun(line string, from, run int) int {
	for i := from; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(line) && line[i] == '`' {
			i++
		}
		if i-start == run {
			return start
		}
	}
	return -1
}

// inCode reports whether offset falls inside any region.
func inCode(regions []region, offset int) bool {
	for _, r := range regions {
		if offset >= r.start && offset < r.end {
			return true
		}
	}
	return false
}
