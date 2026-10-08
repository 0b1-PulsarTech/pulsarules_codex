package projectfile

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

type segment struct {
	name    string
	index   int
	indexed bool
}

type Field struct {
	parent *Field
	seg    segment
	kids   map[segment]*Field
	marked bool
	under  bool
}

func newRoot() *Field {
	// Interned: equal fields are one node, and a "." or "[" in a key never splits a segment.
	return &Field{}
}

func (at *Field) key(name string) *Field {
	if name == "" {
		name = "<empty key>"
	}
	return at.child(segment{name: name})
}

func (at *Field) item(idx int) *Field {
	return at.child(segment{index: idx, indexed: true})
}

func (at *Field) child(seg segment) *Field {
	if next, found := at.kids[seg]; found {
		return next
	}
	if at.kids == nil {
		at.kids = map[segment]*Field{}
	}
	next := &Field{parent: at, seg: seg}
	at.kids[seg] = next
	return next
}

func (at *Field) String() string {
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

func (at *Field) mark() {
	at.marked = true
	for node := at; node.parent != nil; node = node.parent {
		if node.seg.indexed {
			node.parent.under = true
		}
	}
}

func (at *Field) covered(walked *int) bool {
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

func clipValue(value string) string {
	if len(value) <= maxClipBytes {
		return value
	}
	cut := maxClipBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "..."
}
