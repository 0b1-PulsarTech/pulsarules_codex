package character

import "strings"

type region struct {
	start int
	end   int
}

// simplification: a fence is a run of 3+ backticks or tildes closed by an equal-or-longer
// run of the same char; an inline span is a backtick run closed by an equal run on its line.
// Upgrade path: a CommonMark scanner if nested or escaped delimiters give wrong answers.
func codeRegions(src string) []region {
	const minFenceRun = 3

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

func inlineSpans(line string, base int) []region {
	var spans []region
	for pos := 0; pos < len(line); {
		if line[pos] != '`' {
			pos++
			continue
		}
		open := pos
		for pos < len(line) && line[pos] == '`' {
			pos++
		}
		run := pos - open
		if end := closingRun(line, pos, run); end >= 0 {
			spans = append(spans, region{start: base + open, end: base + end + run})
			pos = end + run
		}
	}
	return spans
}

func closingRun(line string, from, run int) int {
	for pos := from; pos < len(line); {
		if line[pos] != '`' {
			pos++
			continue
		}
		start := pos
		for pos < len(line) && line[pos] == '`' {
			pos++
		}
		if pos-start == run {
			return start
		}
	}
	return -1
}

func inCode(regions []region, offset int) bool {
	for _, span := range regions {
		if offset >= span.start && offset < span.end {
			return true
		}
	}
	return false
}
