package markdown

import (
	"sort"
	"strings"
)

type Region struct {
	Start int
	End   int
}

type Code struct {
	certain  []Region
	possible []Region
}

func HasExt(ext string) bool { return strings.EqualFold(ext, ".md") }

func CodeIn(ext, src string) Code {
	// Only Markdown has fenced or indented code.
	if !HasExt(ext) {
		return Code{}
	}
	return codeRegions(src)
}

func (code Code) Certain(offset int) bool { return inRegions(code.certain, offset) }

func (code Code) Possible(offset int) bool { return inRegions(code.possible, offset) }

// simplification: quotes, lists and tabs nest a fence in ways only a full CommonMark parser
// settles, so several readings are scanned side by side. Upgrade path: a pinned goldmark, once
// its code blocks cover Certain and stay inside Possible on a recorded corpus.
func codeRegions(src string) Code {
	scans := make([]*readingScan, 0, len(readings()))
	for _, how := range readings() {
		scans = append(scans, &readingScan{how: how})
	}
	offset := 0
	for _, text := range strings.SplitAfter(src, "\n") {
		for _, scan := range scans {
			scan.line(text, offset)
		}
		offset += len(text)
	}
	lists := make([][]Region, 0, len(scans))
	for _, scan := range scans {
		lists = append(lists, scan.finish(len(src)))
	}
	return Code{certain: intersectAll(lists), possible: unionAll(lists)}
}

func inRegions(regions []Region, offset int) bool {
	// Regions are sorted and disjoint, so the first one ending after offset is the only candidate.
	idx := sort.Search(len(regions), func(at int) bool { return regions[at].End > offset })
	return idx < len(regions) && regions[idx].Start <= offset
}
