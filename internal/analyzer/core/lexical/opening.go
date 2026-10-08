package lexical

import "strings"

func (scan *literalScan) opening(rest string) *StringDelim {
	var best *StringDelim
	for idx := range scan.syntax.Delims {
		delim := &scan.syntax.Delims[idx]
		if scan.opensHere(delim, rest) && outranks(delim, best) {
			best = delim
		}
	}
	return best
}

func (scan *literalScan) opensHere(delim *StringDelim, rest string) bool {
	if !strings.HasPrefix(rest, delim.Open) {
		return false
	}
	if delim.Tag != "" && !scan.tagBefore(delim.Tag) {
		return false
	}
	return !delim.Regex || scan.regexAllowed()
}

func outranks(delim, best *StringDelim) bool {
	if best == nil || len(delim.Open) > len(best.Open) {
		return true
	}
	return len(delim.Open) == len(best.Open) && delim.Tag != "" && best.Tag == ""
}

func (scan *literalScan) tagBefore(tag string) bool {
	// The tag is a whole word right before the opener, blanks and comments aside.
	head := scan.src[:scan.lastEnd]
	if !strings.HasSuffix(head, tag) {
		return false
	}
	before := len(head) - len(tag)
	return before == 0 || !isWordByte(scan.src[before-1])
}
