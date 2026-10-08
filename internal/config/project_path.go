package config

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

type segment struct {
	name    string
	index   int
	indexed bool
}

type fieldPath struct {
	parent *fieldPath
	seg    segment
	kids   map[segment]*fieldPath
	marked bool
	under  bool
}

func newRoot() *fieldPath {
	// Interned: equal fields are one node, and a "." or "[" in a key never splits a segment.
	return &fieldPath{}
}

func (at *fieldPath) key(name string) *fieldPath {
	if name == "" {
		name = "<empty key>"
	}
	return at.child(segment{name: name})
}

func (at *fieldPath) item(idx int) *fieldPath {
	return at.child(segment{index: idx, indexed: true})
}

func (at *fieldPath) child(seg segment) *fieldPath {
	if next, found := at.kids[seg]; found {
		return next
	}
	if at.kids == nil {
		at.kids = map[segment]*fieldPath{}
	}
	next := &fieldPath{parent: at, seg: seg}
	at.kids[seg] = next
	return next
}

func (at *fieldPath) String() string {
	var chain []segment
	for node := at; node.parent != nil; node = node.parent {
		chain = append(chain, node.seg)
	}
	var out strings.Builder
	for _, seg := range slices.Backward(chain) {
		switch {
		case seg.indexed:
			out.WriteString("[" + strconv.Itoa(seg.index) + "]")
		case out.Len() > 0:
			out.WriteString("." + seg.name)
		default:
			out.WriteString(seg.name)
		}
	}
	return out.String()
}

func (at *fieldPath) mark() {
	at.marked = true
	for node := at; node.parent != nil; node = node.parent {
		if node.seg.indexed {
			node.parent.under = true
		}
	}
}

func (at *fieldPath) covered(walked *int) bool {
	// Covered by an equal or enclosing problem, or when it holds an item with a problem.
	if at.marked || at.under {
		return true
	}
	for node := at.parent; node != nil; node = node.parent {
		*walked++
		if node.marked {
			return true
		}
	}
	return false
}

func hasBrace(glob string) bool {
	// A brace inside a [...] class is a literal member, so only those outside one count.
	inClass := false
	for idx := 0; idx < len(glob); idx++ {
		switch char := glob[idx]; {
		case char == '\\':
			idx++
		case inClass:
			inClass = char != ']'
		case char == '[':
			inClass = true
		case char == '{' || char == '}':
			return true
		}
	}
	return false
}

func starCount(glob string) int {
	count := 0
	for part := range strings.SplitSeq(glob, "/") {
		if part != "**" && part != "*" {
			count += strings.Count(part, "*")
		}
	}
	return count
}

func wildcardWork(glob string) int {
	// Each ? and each byte of a [...] class (brackets included) costs path.Match a re-scan.
	work, inClass := 0, false
	for idx := 0; idx < len(glob); idx++ {
		char := glob[idx]
		switch {
		case char == '\\':
			idx++
			if inClass {
				work += 2
			}
		case inClass:
			work++
			inClass = char != ']'
		case char == '[':
			work++
			inClass = true
		case char == '?':
			work++
		}
	}
	return work
}

func starTail(glob string) int {
	// path.Match re-scans the rest of a segment from every start position of a *.
	tail := 0
	for part := range strings.SplitSeq(glob, "/") {
		if star := strings.IndexByte(part, '*'); star >= 0 {
			tail += len(part) - star - strings.Count(part[star:], "*")
		}
	}
	return tail
}

func ClipName(name string, limit int) string {
	if len(escape.Path(name)) <= limit {
		return name
	}
	// Room for the quotes Path may add and for the ellipsis.
	kept, _ := fit(name, limit-len(`""`)-len(ellipsis), tailEdge, quotedWidth)
	return ellipsis + kept
}

func quotedWidth(unit string) int { return len(strconv.Quote(unit)) - len(`""`) }

func clipValue(value string) string {
	if len(value) <= MaxClipBytes {
		return value
	}
	cut := MaxClipBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "..."
}
