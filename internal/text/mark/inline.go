package mark

type tickRun struct {
	start  int
	length int
}

type tickEscaping uint8

const (
	literalTicks tickEscaping = iota
	escapedTicks
)

func inlineSpans(line string, base int, ticks tickEscaping) []Region {
	return pairSpans(tickRuns(line, ticks), base)
}

func pairSpans(runs []tickRun, base int) []Region {
	// A per-length table finds each opener's closer, which keeps unmatched runs linear.
	nextSame := make([]int, len(runs))
	lastOfLength := map[int]int{}
	for idx := len(runs) - 1; idx >= 0; idx-- {
		nextSame[idx] = -1
		if closer, found := lastOfLength[runs[idx].length]; found {
			nextSame[idx] = closer
		}
		lastOfLength[runs[idx].length] = idx
	}
	var spans []Region
	for idx := 0; idx < len(runs); idx++ {
		if closer := nextSame[idx]; closer >= 0 {
			end := runs[closer].start + runs[closer].length
			spans = append(spans, Region{Start: base + runs[idx].start, End: base + end})
			idx = closer
		}
	}
	return spans
}

func tickRuns(line string, ticks tickEscaping) []tickRun {
	var runs []tickRun
	for pos := 0; pos < len(line); {
		if line[pos] != '`' {
			pos++
			continue
		}
		run := tickRun{start: pos}
		for pos < len(line) && line[pos] == '`' {
			pos++
		}
		run.length = pos - run.start
		// An odd count of backslashes escapes the first tick of the run.
		if ticks == escapedTicks && backslashesBefore(line, run.start)%2 == 1 {
			run.start++
			run.length--
		}
		if run.length > 0 {
			runs = append(runs, run)
		}
	}
	return runs
}

func backslashesBefore(line string, at int) int {
	count := 0
	for at-count > 0 && line[at-count-1] == '\\' {
		count++
	}
	return count
}
