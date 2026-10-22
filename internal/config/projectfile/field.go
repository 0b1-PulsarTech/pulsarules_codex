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

func clipValue(value string) string {
	if len(value) <= MaxClipBytes {
		return value
	}
	cut := MaxClipBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + ellipsis
}
