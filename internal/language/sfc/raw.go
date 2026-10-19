package sfc

import (
	"slices"
	"strings"
)

func (scan *markup) raw(name, attrs string) error {
	start := scan.pos
	stop := closeTag(scan.lower[start:], name)
	scan.visit(scanWidth(stop, len(scan.lower)-start))
	record := name == "script" && scan.topLevel() && scan.wanted(attrs)
	scan.noteDropped(name)
	if stop < 0 {
		scan.pos = len(scan.lower)
		scan.note(name, start, len(scan.lower))
		if record {
			return errUnterminated
		}
		return nil
	}
	stop += start
	scan.note(name, start, stop)
	scan.pos = stop
	scan.skipPast(">")
	if record {
		scan.scripts = append(scan.scripts, Script{
			TypeScript: isTypeScript(attrs),
			Text:       blank(scan.text, start, stop),
		})
	}
	return nil
}

func closeTag(rest, name string) int {
	closer := "</" + name
	for from := 0; from < len(rest); {
		at := strings.Index(rest[from:], closer)
		if at < 0 {
			return -1
		}
		after := from + at + len(closer)
		if after == len(rest) || strings.IndexByte(blanks+"/>", rest[after]) >= 0 {
			return from + at
		}
		from += at + 1
	}
	return -1
}

func scanWidth(found, rest int) int {
	if found < 0 {
		return rest
	}
	return found
}

func (scan *markup) wanted(attrs string) bool {
	if !isJavaScript(attrs) {
		return false
	}
	if isModule(attrs) {
		taken := scan.hasModule
		scan.hasModule = true
		return !taken
	}
	taken := scan.hasInstance
	scan.hasInstance = true
	return !taken
}

func (scan *markup) noteDropped(name string) {
	if name == "script" && !scan.topLevel() && !slices.Contains(scan.open, "svelte:head") {
		scan.dropped = true
	}
}

func (scan *markup) note(name string, start, end int) {
	scan.regions = append(scan.regions, Region{Tag: name, Start: start, End: end})
}
