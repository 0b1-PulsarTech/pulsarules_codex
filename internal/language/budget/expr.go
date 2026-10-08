package budget

type Scan struct {
	End       int
	Closed    bool
	Exhausted bool
}

func BraceEnd(text string, from int) Scan {
	scan := newScanner(text)
	scan.pos = from
	scan.push(kindBrace)
	scan.operator('{')
	for scan.pos < len(scan.text) && scan.fault == "" && scan.deepest <= MaxNesting {
		scan.step()
		if len(scan.stack) == 0 {
			return Scan{End: scan.pos, Closed: true}
		}
	}
	return scan.resume()
}

func (scan *scanner) resume() Scan {
	end := min(scan.pos, len(scan.text))
	exhausted := scan.pos >= len(scan.text)
	if exhausted && scan.firstRegex >= 0 {
		end = lineEnd(scan.text, scan.firstRegex)
	}
	return Scan{End: end, Exhausted: exhausted}
}
